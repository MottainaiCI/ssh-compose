/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package specs

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	helpers_sec "github.com/MottainaiCI/ssh-compose/pkg/helpers/security"

	"gopkg.in/yaml.v3"
)

const (
	AuthMethodPassword  = "password"
	AuthMethodPublickey = "publickey"

	// Cisco specific options
	OptionTermHeight   = "height"
	OptionTermWidth    = "width"
	OptionDeadlineSecs = "deadline_secs"
	OptionWaitMs       = "wait_ms"
	OptionBannerLines  = "banner_lines"
)

type RemotesConfig struct {
	File             string             `json:"-" yaml:"-"`
	DefaultRemote    string             `json:"default-remote,omitempty" yaml:"default-remote,omitempty"`
	Remotes          map[string]*Remote `json:"remotes,omitempty" yaml:"remotes,omitempty"`
	Encrypted        bool               `json:"encrypted,omitempty" yaml:"encrypted,omitempty"`
	EncryptedContent string             `json:"enc_content,omitempty" yaml:"enc_content,omitempty"`
}

type Remote struct {
	Host     string `json:"host,omitempty" yaml:"host,omitempty"`
	Port     int    `json:"port,omitempty" yaml:"port,omitempty"`
	Protocol string `json:"protocol,omitempty" yaml:"protocol,omitempty"`
	// See RFC4252. Values: publickey|password
	AuthMethod     string `json:"auth_type,omitempty" yaml:"auth_type,omitempty"`
	PrivateKeyFile string `json:"privatekey_file,omitempty" yaml:"privatekey_file,omitempty"`
	PrivateKeyPass string `json:"privatekey_pass,omitempty" yaml:"privatekey_pass,omitempty"`
	PrivateKeyRaw  string `json:"privatekey_raw,omitempty" yaml:"privatekey_raw,omitempty"`
	User           string `json:"user,omitempty" yaml:"user,omitempty"`
	Pass           string `json:"pass,omitempty" yaml:"pass,omitempty"`
	TimeoutSecs    *uint  `json:"timeout_secs,omitempty" yaml:"timeout_secs,omitempty"`
	// Enable special single session mode for Cisco Device
	CiscoDevice    bool   `json:"cisco_device,omitempty" yaml:"cisco_device,omitempty"`
	CiscoPrompt    string `json:"cisco_prompt,omitempty" yaml:"cisco_prompt,omitempty"`
	CiscoEnaPrompt string `json:"cisco_enaprompt,omitempty" yaml:"cisco_enaprompt,omitempty"`
	CiscoEnaPass   string `json:"cisco_enapass,omitempty" yaml:"cisco_enapass,omitempty"`

	Labels  []string          `json:"labels,omitempty" yaml:"labels,omitempty"`
	Options map[string]string `json:"options,omitempty" yaml:"options,omitempty"`

	Chain []Remote `json:"chain,omitempty" yaml:"chain,omitempty"`
	// Local port for tunnel
	TunLocalPort int    `json:"tun_local_port,omitempty" yaml:"tun_local_port,omitempty"`
	TunLocalAddr string `json:"tun_local_addr,omitempty" yaml:"tun_local_addr,omitempty"`
	TunLocalBind bool   `json:"tun_local_bind,omitempty" yaml:"tun_local_bind,omitempty"`
}

func NewRemote(host, protocol, authMethod string, port int) *Remote {
	return &Remote{
		Host:           host,
		Port:           port,
		Protocol:       protocol,
		AuthMethod:     authMethod,
		PrivateKeyFile: "",
		PrivateKeyPass: "",
		PrivateKeyRaw:  "",
		User:           "",
		Pass:           "",
		TimeoutSecs:    nil,
		TunLocalBind:   false,
	}
}

func (r *Remote) SetPrivateKeyFile(f string)  { r.PrivateKeyFile = f }
func (r *Remote) SetPrivateKeyPass(p string)  { r.PrivateKeyPass = p }
func (r *Remote) SetPrivateKeyRaw(p string)   { r.PrivateKeyRaw = p }
func (r *Remote) SetUser(u string)            { r.User = u }
func (r *Remote) SetPass(p string)            { r.Pass = p }
func (r *Remote) SetTunLocalPort(port int)    { r.TunLocalPort = port }
func (r *Remote) SetTimeoutSecs(timeout uint) { r.TimeoutSecs = &timeout }

func (r *Remote) GetHost() string           { return r.Host }
func (r *Remote) GetPort() int              { return r.Port }
func (r *Remote) GetProtocol() string       { return r.Protocol }
func (r *Remote) GetAuthMethod() string     { return r.AuthMethod }
func (r *Remote) GetPrivateKeyFile() string { return r.PrivateKeyFile }
func (r *Remote) GetPrivateKeyPass() string { return r.PrivateKeyPass }
func (r *Remote) GetPrivateKeyRaw() string  { return r.PrivateKeyRaw }
func (r *Remote) GetUser() string           { return r.User }
func (r *Remote) GetPass() string           { return r.Pass }
func (r *Remote) GetTimeoutSecs() *uint     { return r.TimeoutSecs }
func (r *Remote) GetTunLocalPort() int      { return r.TunLocalPort }
func (r *Remote) GetTunLocalBind() bool     { return r.TunLocalBind }
func (r *Remote) GetTunLocalAddr() string   { return r.TunLocalAddr }
func (r *Remote) GetCiscoPrompt() string    { return r.CiscoPrompt }
func (r *Remote) GetCiscoEnaPrompt() string { return r.CiscoEnaPrompt }
func (r *Remote) GetCiscoEnaPass() string   { return r.CiscoEnaPass }
func (r *Remote) GetCiscoDevice() bool      { return r.CiscoDevice }
func (r *Remote) GetChain() []Remote        { return r.Chain }

func (r *Remote) HasChain() bool { return len(r.Chain) > 0 }

func (r *Remote) GetOption(o string) string {
	if r.Options != nil {
		for k, v := range r.Options {
			if k == o {
				return v
			}
		}
	}
	return ""
}
func (r *Remote) HasLabel(l string) bool {
	ans := false
	if r.Labels != nil {
		for _, label := range r.Labels {
			if label == l {
				ans = true
				break
			}
		}
	}
	return ans
}

func (r *Remote) GetLabels() []string           { return r.Labels }
func (r *Remote) GetOptions() map[string]string { return r.Options }

func (r *Remote) Sanitize() {
	if r.GetProtocol() == "" {
		r.Protocol = "tcp"
	}
	if r.GetAuthMethod() == "" {
		r.AuthMethod = "publickey"
	}

	if r.HasChain() {
		for t := range r.Chain {
			r.Chain[t].Sanitize()
		}
	}
}

func NewRemotesConfig() *RemotesConfig {
	return &RemotesConfig{
		File:          "",
		DefaultRemote: "",
		Remotes:       make(map[string]*Remote, 0),
	}
}

func RemotesConfigFromYaml(data []byte, file string) (*RemotesConfig, error) {
	ans := &RemotesConfig{}
	if err := yaml.Unmarshal(data, ans); err != nil {
		return nil, err
	}
	ans.File = file

	return ans, nil
}

func RemotesConfigFromFile(file string, config *SshComposeConfig) (*RemotesConfig, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	ans, err := RemotesConfigFromYaml(content, file)
	if err != nil {
		return nil, err
	}

	if ans.Encrypted {
		if config.GetSecurity().Key == "" {
			return nil, fmt.Errorf("Found remotes encrypted but no key defined!")
		}
		keyBytes, err := base64.StdEncoding.DecodeString(config.GetSecurity().Key)
		if err != nil {
			return nil, fmt.Errorf("error on decode base64 key: %s", err.Error())
		}

		// Decode encrypted content.
		encryptedContent, err := base64.StdEncoding.DecodeString(
			ans.EncryptedContent,
		)
		if err != nil {
			return nil, fmt.Errorf("error on decode base64 for file %s:\n%s",
				file, err.Error())
		}

		dkaOpts := helpers_sec.NewDKAOptsDefault()
		if config.GetSecurity().DKAOpts != nil {
			if config.GetSecurity().DKAOpts.TimeIterations != nil {
				dkaOpts.TimeIterations = *config.GetSecurity().DKAOpts.TimeIterations
			}
			if config.GetSecurity().DKAOpts.MemoryUsage != nil {
				dkaOpts.MemoryUsage = *config.GetSecurity().DKAOpts.MemoryUsage
			}
			if config.GetSecurity().DKAOpts.KeyLength != nil {
				dkaOpts.KeyLength = *config.GetSecurity().DKAOpts.KeyLength
			}
			if config.GetSecurity().DKAOpts.Parallelism != nil {
				dkaOpts.Parallelism = *config.GetSecurity().DKAOpts.Parallelism
			}
		}
		decodedBytes, err := helpers_sec.Decrypt(encryptedContent, keyBytes, dkaOpts)
		if err != nil {
			return nil, fmt.Errorf("ignoring error on decrypt content of the file %s: %s",
				file, err.Error())
		}

		remotesDecrypted, err := RemotesConfigFromYaml(decodedBytes, file)
		if err != nil {
			return nil, fmt.Errorf("error on parse decoded content of the file %s: %s",
				file, err.Error())
		}

		ans.Remotes = remotesDecrypted.Remotes
		ans.DefaultRemote = remotesDecrypted.DefaultRemote
		ans.EncryptedContent = ""
	}

	return ans, nil
}

func (rc *RemotesConfig) SetDefault(remote string) { rc.DefaultRemote = remote }
func (rc *RemotesConfig) GetDefault() string       { return rc.DefaultRemote }

func (rc *RemotesConfig) HasRemote(remote string) bool {
	if _, present := rc.Remotes[remote]; present {
		return true
	}
	return false
}

func (rc *RemotesConfig) GetRemote(remote string) *Remote {
	if ans, present := rc.Remotes[remote]; present {
		return ans
	}
	return nil
}

func (rc *RemotesConfig) Sanitize() {
	for k := range rc.Remotes {
		rc.Remotes[k].Sanitize()
	}
}

func (rc *RemotesConfig) AddRemote(name string, r *Remote) {
	rc.Remotes[name] = r
}

func (rc *RemotesConfig) DelRemote(name string) {
	delete(rc.Remotes, name)
}

func (rc *RemotesConfig) Write(config *SshComposeConfig) error {
	if rc.File == "" {
		return fmt.Errorf("Remotes Config without file path")
	}

	data, err := yaml.Marshal(rc)
	if err != nil {
		return err
	}

	if rc.Encrypted {

		if config.GetSecurity().Key == "" {
			return fmt.Errorf("no key available to write remotes config")
		}

		keyBytes, err := base64.StdEncoding.DecodeString(config.GetSecurity().Key)
		if err != nil {
			fmt.Println("error on decode key: %s", err.Error())
			os.Exit(1)
		}

		dkaOpts := helpers_sec.NewDKAOptsDefault()
		if config.GetSecurity().DKAOpts != nil {
			if config.GetSecurity().DKAOpts.TimeIterations != nil {
				dkaOpts.TimeIterations = *config.GetSecurity().DKAOpts.TimeIterations
			}
			if config.GetSecurity().DKAOpts.MemoryUsage != nil {
				dkaOpts.MemoryUsage = *config.GetSecurity().DKAOpts.MemoryUsage
			}
			if config.GetSecurity().DKAOpts.KeyLength != nil {
				dkaOpts.KeyLength = *config.GetSecurity().DKAOpts.KeyLength
			}
			if config.GetSecurity().DKAOpts.Parallelism != nil {
				dkaOpts.Parallelism = *config.GetSecurity().DKAOpts.Parallelism
			}
		}
		encryptedContent, err := helpers_sec.Encrypt(data, keyBytes, dkaOpts)
		if err != nil {
			fmt.Println(fmt.Sprintf("Error on encrypt content of the file %s: %s",
				rc.File, err.Error()))
			os.Exit(1)
		}

		// Create a new RemotesConfig where store encrypted data
		erc := NewRemotesConfig()
		erc.Encrypted = true
		erc.EncryptedContent = base64.StdEncoding.EncodeToString(encryptedContent)

		data, err = yaml.Marshal(erc)
		if err != nil {
			return err
		}

	}

	return os.WriteFile(rc.File, data, 0644)
}

func (rc *RemotesConfig) GetAbsConfigDir() (string, error) {
	ans := ""
	if rc.File == "" {
		return "", fmt.Errorf("Remotes Config without file path")
	}

	if strings.HasPrefix(rc.File, "/") {
		ans = filepath.Dir(rc.File)
	} else {
		abs, err := filepath.Abs(rc.File)
		if err != nil {
			return "", err
		}

		ans = filepath.Dir(abs)
	}

	return ans, nil
}
