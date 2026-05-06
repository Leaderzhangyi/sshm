package ssh

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"sshm/internal/config"
)

type ProgressCallback func(percent float64)

func newSSHClient(c *config.Connection) (*ssh.Client, error) {
	cfg := &ssh.ClientConfig{
		User:            c.User,
		Auth:            []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	if c.Auth == "key" && c.KeyPath != "" {
		key, err := os.ReadFile(config.ExpandPath(c.KeyPath))
		if err != nil {
			return nil, fmt.Errorf("读取私钥失败: %w", err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("解析私钥失败: %w", err)
		}
		cfg.Auth = append(cfg.Auth, ssh.PublicKeys(signer))
	} else if c.Password != "" {
		cfg.Auth = append(cfg.Auth, ssh.Password(c.Password))
	}

	var client *ssh.Client
	var err error

	if c.ProxyJump != "" {
		jumpUser, jumpHost, jumpPort := config.ParseJumpHost(c.ProxyJump)
		jumpAddr := fmt.Sprintf("%s:%d", jumpHost, jumpPort)

		jumpCfg := &ssh.ClientConfig{
			User: jumpUser,
			Auth: []ssh.AuthMethod{
				ssh.PasswordCallback(func() (string, error) {
					return "", nil
				}),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         30 * time.Second,
		}

		jumpClient, err := ssh.Dial("tcp", jumpAddr, jumpCfg)
		if err != nil {
			return nil, fmt.Errorf("连接跳板机失败: %w", err)
		}

		targetAddr := fmt.Sprintf("%s:%d", c.Host, c.Port)
		if c.Port == 0 {
			targetAddr = fmt.Sprintf("%s:22", c.Host)
		}

		conn, err := jumpClient.Dial("tcp", targetAddr)
		if err != nil {
			jumpClient.Close()
			return nil, fmt.Errorf("通过跳板机连接目标失败: %w", err)
		}

		ncc, chans, reqs, err := ssh.NewClientConn(conn, targetAddr, cfg)
		if err != nil {
			conn.Close()
			jumpClient.Close()
			return nil, fmt.Errorf("建立SSH连接失败: %w", err)
		}

		client = ssh.NewClient(ncc, chans, reqs)
	} else {
		addr := fmt.Sprintf("%s:%d", c.Host, c.Port)
		if c.Port == 0 {
			addr = fmt.Sprintf("%s:22", c.Host)
		}
		client, err = ssh.Dial("tcp", addr, cfg)
		if err != nil {
			return nil, fmt.Errorf("连接失败: %w", err)
		}
	}

	return client, nil
}

func Upload(c *config.Connection, localPath, remotePath string, progress ProgressCallback) error {
	client, err := newSSHClient(c)
	if err != nil {
		return err
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("创建SFTP客户端失败: %w", err)
	}
	defer sftpClient.Close()

	localPath = config.ExpandPath(localPath)

	info, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("本地路径不存在: %w", err)
	}

	if info.IsDir() {
		return uploadDirectory(sftpClient, localPath, remotePath, progress)
	}
	return uploadFile(sftpClient, localPath, remotePath, progress)
}

func uploadFile(sftpClient *sftp.Client, localPath, remotePath string, progress ProgressCallback) error {
	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("打开本地文件失败: %w", err)
	}
	defer localFile.Close()

	info, _ := localFile.Stat()
	totalSize := info.Size()

	remoteDir := filepath.Dir(remotePath)
	if err := sftpClient.MkdirAll(remoteDir); err != nil {
		return fmt.Errorf("创建远程目录失败: %w", err)
	}

	remoteFile, err := sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("创建远程文件失败: %w", err)
	}
	defer remoteFile.Close()

	buf := make([]byte, 32*1024)
	var copied int64
	for {
		n, err := localFile.Read(buf)
		if n > 0 {
			if _, err := remoteFile.Write(buf[:n]); err != nil {
				return fmt.Errorf("写入远程文件失败: %w", err)
			}
			copied += int64(n)
			if progress != nil && totalSize > 0 {
				progress(float64(copied) / float64(totalSize))
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取本地文件失败: %w", err)
		}
	}

	return nil
}

func uploadDirectory(sftpClient *sftp.Client, localPath, remotePath string, progress ProgressCallback) error {
	if err := sftpClient.MkdirAll(remotePath); err != nil {
		return fmt.Errorf("创建远程目录失败: %w", err)
	}

	return filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(localPath, path)
		remoteFilePath := filepath.Join(remotePath, relPath)

		if info.IsDir() {
			return sftpClient.MkdirAll(remoteFilePath)
		}
		return uploadFile(sftpClient, path, remoteFilePath, progress)
	})
}

func Download(c *config.Connection, remotePath, localPath string, progress ProgressCallback) error {
	client, err := newSSHClient(c)
	if err != nil {
		return err
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("创建SFTP客户端失败: %w", err)
	}
	defer sftpClient.Close()

	localPath = config.ExpandPath(localPath)

	info, err := sftpClient.Stat(remotePath)
	if err != nil {
		return fmt.Errorf("远程路径不存在: %w", err)
	}

	if info.IsDir() {
		return downloadDirectory(sftpClient, remotePath, localPath, progress)
	}
	return downloadFile(sftpClient, remotePath, localPath, progress)
}

func downloadFile(sftpClient *sftp.Client, remotePath, localPath string, progress ProgressCallback) error {
	remoteFile, err := sftpClient.Open(remotePath)
	if err != nil {
		return fmt.Errorf("打开远程文件失败: %w", err)
	}
	defer remoteFile.Close()

	info, _ := remoteFile.Stat()
	totalSize := info.Size()

	localDir := filepath.Dir(localPath)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return fmt.Errorf("创建本地目录失败: %w", err)
	}

	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("创建本地文件失败: %w", err)
	}
	defer localFile.Close()

	buf := make([]byte, 32*1024)
	var copied int64
	for {
		n, err := remoteFile.Read(buf)
		if n > 0 {
			if _, err := localFile.Write(buf[:n]); err != nil {
				return fmt.Errorf("写入本地文件失败: %w", err)
			}
			copied += int64(n)
			if progress != nil && totalSize > 0 {
				progress(float64(copied) / float64(totalSize))
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取远程文件失败: %w", err)
		}
	}

	return nil
}

func downloadDirectory(sftpClient *sftp.Client, remotePath, localPath string, progress ProgressCallback) error {
	if err := os.MkdirAll(localPath, 0755); err != nil {
		return fmt.Errorf("创建本地目录失败: %w", err)
	}

	walker := sftpClient.Walk(remotePath)
	for walker.Step() {
		if err := walker.Err(); err != nil {
			return err
		}

		relPath, _ := filepath.Rel(remotePath, walker.Path())
		localFilePath := filepath.Join(localPath, relPath)

		if walker.Stat().IsDir() {
			if err := os.MkdirAll(localFilePath, 0755); err != nil {
				return err
			}
		} else {
			if err := downloadFile(sftpClient, walker.Path(), localFilePath, progress); err != nil {
				return err
			}
		}
	}

	return nil
}
