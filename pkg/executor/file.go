/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package executor

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	log "github.com/MottainaiCI/ssh-compose/pkg/logger"
)

func (s *SshCExecutor) RecursiveMkdir(dir string, mode *os.FileMode, uid int, gid int, ensurePerms bool) error {
	var err error

	if s.SftpClient == nil {
		return fmt.Errorf("Sftp client not initialized.")
	}

	// special case, every node has /, we don't need to do anything
	if dir == "/" {
		return nil
	}

	pclean := filepath.Clean(dir)
	parts := strings.Split(pclean, "/")
	i := len(parts)

	for ; i >= 1; i-- {
		cur := filepath.Join(parts[:i]...)
		fi, err := s.SftpClient.Stat(cur)
		if err != nil {
			continue
		}

		if !fi.IsDir() {
			return fmt.Errorf("%s is not a directory", cur)
		}

		i++
		break
	}

	for ; i <= len(parts); i++ {
		cur := filepath.Join(parts[:i]...)
		if cur == "" {
			continue
		}

		cur = "/" + cur

		// SFTP goes in error if the directory is already present.
		// I need to check if it's already present.
		fi, _ := s.SftpClient.Stat(cur)
		if fi != nil {
			if !fi.IsDir() {
				return fmt.Errorf("%s is already present but is not a directory",
					cur)
			} else {
				s.Emitter.DebugLog(false,
					fmt.Sprintf("Directory %s already present.", cur))
				continue
			}
		}

		err = s.SftpClient.Mkdir(cur)
		if err != nil {
			return err
		}

		s.Emitter.DebugLog(false, fmt.Sprintf("Creating %s (%s)", cur, "directory"))

		if ensurePerms {
			err = s.SftpClient.Chmod(cur, *mode)
			if err != nil {
				return err
			}

			err = s.SftpClient.Chown(cur, uid, gid)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *SshCExecutor) RecursivePushFile(nodeName, source, target string, ensurePerms bool) error {

	var targetIsFile bool = true
	var sourceIsFile bool = true
	var uid, gid int
	var mode os.FileMode

	if strings.HasSuffix(source, "/") {
		sourceIsFile = false
	}

	if strings.HasSuffix(target, "/") {
		targetIsFile = false
	}

	dir := filepath.Dir(target)
	sourceDir := filepath.Dir(filepath.Clean(source))
	if !sourceIsFile && targetIsFile {
		dir = target
		sourceDir = source
	}
	sourceLen := len(sourceDir)

	fi, err := os.Stat(sourceDir)
	if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
		uid = int(stat.Uid)
		gid = int(stat.Gid)
		mode = fi.Mode().Perm()
	} else {
		// Using os uid/gid
		uid = os.Getuid()
		gid = os.Getgid()
		mode = os.FileMode(0755)
	}
	err = s.RecursiveMkdir(dir, &mode, uid, gid, ensurePerms)
	if err != nil {
		return errors.New("Error on create dir " + filepath.Dir(target) + ": " + err.Error())
	}

	sendFile := func(p string, fInfo os.FileInfo, err error) error {

		if err != nil {
			return fmt.Errorf("failed to walk path for %s: %s", p, err)
		}

		// Detect unsupported files
		if !fInfo.Mode().IsRegular() && !fInfo.Mode().IsDir() && fInfo.Mode()&os.ModeSymlink != os.ModeSymlink {
			return fmt.Errorf("'%s' isn't a supported file type", p)
		}

		// Prepare for file transfer
		targetPath := path.Join(target, filepath.ToSlash(p[sourceLen:]))

		if p == source {
			if targetIsFile && sourceIsFile {
				targetPath = target
			} else if targetIsFile && !sourceIsFile {
				// Nothing to do. The directory is already been created.
				s.Emitter.DebugLog(false, fmt.Sprintf("Skipping dir %s. Already created.", p))
				return nil
			}
		}

		if stat, ok := fInfo.Sys().(*syscall.Stat_t); ok {
			uid = int(stat.Uid)
			gid = int(stat.Gid)
		} else {
			// Using os uid/gid
			uid = os.Getuid()
			gid = os.Getgid()
		}

		logger := log.GetDefaultLogger()

		ftype := "file"
		if fInfo.IsDir() {
			// Directory handling
			err = s.RecursiveMkdir(targetPath, &mode, uid, gid, ensurePerms)
			if err != nil {
				return err
			}
			ftype = "directory"

		} else if fInfo.Mode()&os.ModeSymlink == os.ModeSymlink {
			// Symlink handling
			symlinkTarget, err := os.Readlink(p)
			if err != nil {
				return err
			}

			err = s.SftpClient.Symlink(symlinkTarget, targetPath)
			if err != nil {
				return err
			}

			ftype = "symlink"
		} else {
			// Open local file for reading data
			f, err := os.Open(p)
			if err != nil {
				return err
			}
			defer f.Close()

			// Open target file for writing data
			dstFile, err := s.SftpClient.Create(targetPath)
			if err != nil {
				return err
			}
			defer dstFile.Close()

			_, err = io.Copy(dstFile, f)
			if err != nil {
				return err
			}
		}

		if ensurePerms {
			err = s.SftpClient.Chmod(targetPath, mode)
			if err != nil {
				return err
			}

			err = s.SftpClient.Chown(targetPath, uid, gid)
			if err != nil {
				return err
			}
		}

		if logger.Config.GetGeneral().Debug {
			s.Emitter.InfoLog(true,
				logger.Aurora.Italic(
					logger.Aurora.BrightMagenta(
						fmt.Sprintf(">>> [%s] Pushing %s -> %s (%s)",
							nodeName, p, targetPath, ftype))))
		}

		return nil
	}

	return filepath.Walk(source, sendFile)
}

func (s *SshCExecutor) RecursivePullFile(nodeName, sourcePath, targetPath string, localAsTarget, ensurePerms bool) error {
	var err error
	var ftype string
	var uid, gid int
	var mode os.FileMode

	// Retrieve the information of the remote source directory
	fi, err := s.SftpClient.Stat(sourcePath)
	if err != nil {
		return err
	}

	ftype = "file"
	if fi.IsDir() {
		ftype = "directory"
	} else if fi.Mode()&os.ModeSymlink != 0 {
		ftype = "symlink"
	}
	mode = fi.Mode()
	if ensurePerms {
		if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
			uid = int(stat.Uid)
			gid = int(stat.Gid)
		} else {
			// Using os uid/gid
			uid = os.Getuid()
			gid = os.Getgid()
		}
	} else {
		uid = os.Getuid()
		gid = os.Getgid()
	}

	var target string
	// Default logic is to append tree to target directory
	if localAsTarget {
		target = targetPath
	} else {
		target = filepath.Join(targetPath, sourcePath)
	}

	logger := log.GetDefaultLogger()
	if logger.Config.GetGeneral().Debug {
		s.Emitter.InfoLog(true,
			logger.Aurora.Italic(
				fmt.Sprintf(">>> [%s] Pulling %s -> %s (%s) (%v) - (%s)",
					nodeName, sourcePath, targetPath, target, localAsTarget, ftype)))
	}

	if ftype == "directory" {

		err := os.MkdirAll(target, mode)
		if err != nil {
			s.Emitter.InfoLog(false, fmt.Sprintf("directory %s is already present. Nothing to do.\n", target))
		}

		if ensurePerms {
			err = os.Chown(target, uid, gid)
			if err != nil {
				return err
			}
		}

		// Retrieve the list of the entries of the source directory
		entries, err := s.SftpClient.ReadDir(sourcePath)
		if err != nil {
			return err
		}

		for _, ent := range entries {
			nextP := path.Join(sourcePath, ent.Name())
			nextT := path.Join(target, ent.Name())
			err = s.RecursivePullFile(nodeName, nextP, nextT, true, ensurePerms)
			if err != nil {
				return err
			}
		}
	} else if ftype == "file" {

		// Open the local file
		f, err := os.Create(target)
		if err != nil {
			return err
		}
		defer f.Close()

		err = os.Chmod(target, mode)
		if err != nil {
			return err
		}

		// open the remote source file
		sfile, err := s.SftpClient.Open(sourcePath)
		if err != nil {
			return err
		}
		defer sfile.Close()

		_, err = io.Copy(f, sfile)
		if err != nil {
			s.Emitter.ErrorLog(false, fmt.Sprintf("Error on pull file %s", target))
			return err
		}

	} else if ftype == "symlink" {
		// Read fileinfo of the link
		fiLink, err := s.SftpClient.Lstat(sourcePath)
		if err != nil {
			return err
		}

		err = os.Symlink(fiLink.Name(), target)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("Unknown file type '%s'", ftype)
	}

	return nil
}
