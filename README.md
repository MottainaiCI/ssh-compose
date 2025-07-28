# SSH Compose

[![Build Status](https://github.com/MottainaiCI/ssh-compose/actions/workflows/push.yml/badge.svg)](https://github.com/MottainaiCI/ssh-compose/actions/workflows/push.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/MottainaiCI/ssh-compose)](https://goreportcard.com/report/github.com/MottainaiCI/ssh-compose)
[![CodeQL](https://github.com/MottainaiCI/ssh-compose/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/MottainaiCI/ssh-compose/actions/workflows/codeql-analysis.yml)

**ssh-compose** is the sister of the **lxd-compose** project and supplies a way to deploy a complex environment
using SSH protocol.

It permits to organize and trace all configuration steps of infrastructure and create test suites.

All configuration files could be created at runtime through two different template engines: Mottainai or Jinja2 (require `j2cli` tool).

It's under heavy development phase and specification could be changed in the near future.

## Documentation

Incoming...

## Installation

To install `ssh-compose` that has only one statically linked binary you can:

* download it from last release page of Github

* using `anise` and having a way to easily upgrade it when needed:

```
$> curl https://raw.githubusercontent.com/geaaru/luet/geaaru/contrib/config/get_anise_root.sh | sh
$> sudo anise install -y ssh-compose
$> sudo anise cleanup
$> # To upgrade later the package
$> sudo anise upgrade -y
```

## Getting Started

The file `.ssh-compose.yml` contains the rules about `ssh-compose` reads and loads the environments and projects.

Hereinafter, a simple example of the `.ssh-compose.yml` file:

```yaml
general:
  debug: false
  remotes_confdir: ./remotes/

logging:
  level: "info"
  runtime_cmds_output: true

render_values_file: ./render/values.yml
render_default_file: ./render/default.yml
# Define the directories with environments files.
env_dirs:
- ./envs

render_templates_dirs:
- ./render/templates
```

where `./render` directory contains Helm templates files and `default.yml` and `values.yml`
used by `ssh-compose` to generate the final YAML file.

The directory `./remotes` contains the `config.yml` file with the list of the nodes
used in the environment files.
Similarly to the Incus/LXD projects *ssh-compose* uses a config.yml file to define the remotes to reach and the way to connect.
The file `config.yml` is read from `$HOME/.config/ssh-compose/` directory or in the path defined in the file .ssh-compose.yml,
or again through the environment variable `SSHC_CONF`.

Hereinafter, an example of the remotes config file:

```yaml
default-remote: test
remotes:
    mynode2:
        host: 10.20.20.10
        port: 22
        protocol: tcp
        auth_type: publickey
        privatekey_file: /home/geaaru/.ssh/id_ed25519
        user: geaaru
    mynode1:
        host: 10.10.10.10
        port: 22
        auth_type: password
        user: geaaru
        pass: pass
    test:
        host: 172.18.10.192
        port: 22
        protocol: tcp
        auth_type: password
        timeout_secs: 30
        user: root
        pass: pass
```

The `default-remote` is used as default if the `entrypoint` of the node is not defined or in the command
`ssh-compose shell` without other options.

On running remote command `ssh-compose` try to set in the session the enviroments:

- `SSH_COMPOSE_VERSION`: with the version of the ssh-compose used
- `SSH_COMPOSE_PROJECT`: that contains the JSON object with the project environments.

These variables will be availables only if in the target server the SSH Daemon config contains:

```
# cat /etc/ssh/sshd_config  | grep AcceptEnv | grep SSH_COM
AcceptEnv SSH_COMPOSE_*

```

The `SSH_COMPOSE` prefix could be replaced with a custom value defined in the `.ssh-compose.yml` file.

### SSL Tunneling Chain

In order to reach a specific remote over multiple hop it's possible define a chain of node to use and
optional binding a local port to use with other tools.

Every hop node is configurable at the same mode of a normal node.

Following an example of a remote with 3 hop:

```yaml

    test:
        host: 172.18.10.192
        port: 22
        protocol: tcp
        auth_type: password
        user: root
        pass: pass
        chain:
            - host: 10.10.10.1
              port: 22
              protocol: tcp
              auth_type: publickey
              privatekey_file: /home/geaaru/.ssh/id_ed25519
              user: geaaru
            - host: 10.10.20.1
              port: 22
              protocol: tcp
              auth_type: password
              user: user2
              pass: mypass
            - host: 10.10.30.1
              port: 22
              protocol: tcp
              auth_type: publickey
              privatekey_file: /home/geaaru/.ssh/id_ed25519
              user: user3
```

In the example the node 172.18.10.192 (port 22) is reachable through three hops:
  - hop1: geaaru@10.10.10.1:22
  - hop2: user2@10.10.20.1:22
  - hop3: user3@10.10.30.1:22

The SSL channel created over the hop3 is later used to reach 172.18.10.192.

It's also possible define a local port that is used to manage connection to the
final node:

```
```yaml

    test:
        host: 172.18.10.192
        port: 22
        protocol: tcp
        auth_type: password
        user: root
        pass: pass
        # Define the local port used as bridge of the all hops.
        # If this option is not defined the local port is allocated
        # dynamically.
        tun_local_port: 20000
        # Define the local address (localhost, keep empty for all interfaces)
        tun_local_addr: "localhost"
        # Enable local binding.
        tun_local_bind: true
        chain:
            - host: 10.10.10.1
              port: 22
              protocol: tcp
              auth_type: publickey
              privatekey_file: /home/geaaru/.ssh/id_ed25519
              user: geaaru
            - host: 10.10.20.1
              port: 22
              protocol: tcp
              auth_type: password
              user: user2
              pass: mypass
            - host: 10.10.30.1
              port: 22
              protocol: tcp
              auth_type: publickey
              privatekey_file: /home/geaaru/.ssh/id_ed25519
              user: user3
```

The information about hops are visible in debug mode:

```
$ ssh-compose shell test --debug --without-envs
[test] Connecting to first hop at 10.10.10.1:22...
[test] Connecting to hop 2 at 10.10.20.1:22...
[test] Connecting to hop 3 at 10.10.30.1:22...
[test] Binding local tunnel at localhost:20000...
[test] Connecting at 127.0.0.1:20000 to reach 172.18.10.192:22...
test ~ #
```

### Cisco Devices

It's possible to use ssh-compose projects to run hooks over
Cisco device but with limitations.

Cisco Device 3750, for example, doesn't support multi SSH sessions and
avoiding to open and close tons of sockets to device the only possible
way is to use one single session as PTY and write commands through stdin.
This means that it uses ssh.Shell session without having a result value
of the executed command. More complex scenario could be created using
ssh-compose Golang API with a specific application.

So, using the options `cisco_device` and `cisco_prompt`:
```
    cisco-3750:
        host: 10.10.50.1
        port: 22
        protocol: tcp
        auth_type: password
        user: pix
        pass: cisco
        tun_local_port: 20000
        tun_local_addr: "localhost"
        tun_local_bind: false
        cisco_device: true
        # Keep empty to automatically catch the prompt at login
        cisco_prompt: 3750-MYDEV#
```
it's possible to execute commands to a cisco device and store the output in the logfile.

Example:

```bash
$> ssh-compose a cisco-example
>>> Applying project ➡ cisco-example1 🚀 
>>> [node1] - show vrf link-A - ☕ 
  Name                             Default RD            Protocols   Interfaces
  link-A                           1000:1000             ipv4        Vl200
                                                                     Vl500
                                                                     Vl1001
                                                                     Vl1002
3750-MYDEV#
>>> [node1] - show vrf link-B - ☕ 
  Name                             Default RD            Protocols   Interfaces
  link-B                           2000:2000             ipv4        Lo60
                                                                     Lo103
                                                                     Lo104
                                                                     Tu60
                                                                     Tu114
                                                                     Tu115
                                                                     Vl201
                                                                     Vl2194
3750-MYDEV#
🎉 All done!

```

## A simple example

Considering a simple example where I want to upgrade the Ubuntu and Macaroni OS nodes
and configure a specific Nginx installation, these are the steps to follow.

We have 2 nodes with Macaroni OS and 1 node with Ubuntu. To simplify the example, the SSH
connection uses `root` user that normally will be not enabled for security reasons.

1. Creating `.ssh-compose.yml` file

```bash
$> echo "
general:
  debug: false
  remotes_confdir: ./remotes/

logging:
  level: "info"
  runtime_cmds_output: true

render_values_file: ./render/values.yml
render_default_file: ./render/default.yml
# Define the directories with environments files.
env_dirs:
- ./envs

render_templates_dirs:
- ./render/templates
" > .ssh-compose.yml
```

2. Creating needed directories

```bash
$> mkdir -p ./remotes ./envs/hooks ./envs/vars ./render/templates
```

3. Setup remotes `config.yml`

```bash
$> ssh-compose remote add --auth-method password --host 10.10.10.10 \
        --user root --pass mypass --default macaroni01
🎉  Remote macaroni01 created.
$> ssh-compose remote add --auth-method password --host 10.10.10.11 \
        --user root --pass mypass --default macaroni01
🎉  Remote macaroni02 created.
$> ssh-compose remote add --auth-method password --host 10.10.10.12 \
        --user root --pass mypass --default ubuntu01
🎉  Remote ubuntu01 created.

$> ssh-compose remote list
|        NAME        |         URL         | AUTHMETHOD | USER |
|--------------------|---------------------|------------|------|
| macaroni01         | tcp::10.10.10.10:22 | password   | root |
| macaroni02         | tcp::10.10.10.11:22 | password   | root |
| ubuntu01 (default) | tcp::10.10.10.12:22 | password   | root |
```

4. Create the hooks

NOTE: In the example we create splitted hooks but could be defined
directly inside the environment file.

```bash
$> # Create hook with Macaroni OS upgrade commands
$> echo "
hooks:
- event: pre-node-sync
  flags:
    - upgrade
  commands:
    - >-
      anise repo update &&
      anise upgrade -y
" > envs/hooks/macaroni-upgrade.yml

$> # Create hook with Ubuntu upgrade commands
$> echo "
hooks:
- event: pre-node-sync
  flags:
    - upgrade
  commands:
    - >-
      apt-get update &&
      apt-get upgrade -y
" > envs/hooks/ubuntu-upgrade.yml

$> # Create hook to restart NGINX service
$> echo '
hooks:
- event: post-node-sync
  flags:
    - upgrade
  commands:
    - >-
      ubuntu=$(cat /etc/os-release | grep ID| grep ubuntu | wc -l);
      if [ "${ubuntu}" == "1" ] ; then
      systemctl restart nginx ;
      else
      /etc/init.d/nginx restart ; fi
' > envs/hooks/nginx-restart.yml
```

5. Create the render file to better organize the environment file

NOTE: Using the render to organize the environment file is
an option. You can write directly the YAML in the environment.

```bash
$> echo '
nginx_nodes:
- name: "nginx1"
  endpoint: "macaroni01"
  os: "macaroni"
- name: "nginx2"
  endpoint: "macaroni02"
  os: "macaroni"
- name: "nginx3"
  endpoint: "ubuntu01"
  os: "ubuntu"
' > render/values.yml

```

6. Create my NGINX config to sync

Under the path `envs/files/nginx.conf` create the NGINX config of our servers
based on your needs.

7. Create the environment file

```bash
$> echo '
version: "1"
template_engine:
  engine: "mottainai"

projects:
  - name: "nginx-servers"
    description: |-
      Configure my NGINX servers

    groups:
      - name: nginx-group
        description: "NGINX Group Nodes"

        nodes:
{{ range $k, $v := .Values.nginx_nodes }}
          - name: {{ $v.name }}
            endpoint: {{ $v.endpoint }}

            sync_resources:
              - source: files/nginx.conf
                dst: /etc/nginx/nginx.conf

            include_hooks_files:
            {{- if eq $v.os "macaroni" }}
            - hooks/macaroni-upgrade.yml
            {{- else }}
            - hooks/ubuntu-upgrade.yml
            {{- end }}
            - hooks/nginx-restart.yml
{{ end }}
' > envs/nginx-servers.yml
```

If all works fine the project will be visible with:

```bash
$> ssh-compose project list
| PROJECT NAME  |        DESCRIPTION         | # GROUPS |
|---------------|----------------------------|----------|
| nginx-servers | Configure my NGINX servers |        1 |
```

8. Apply the project!

In the example I report only a part of the output.

```bash
$> ssh-compose a nginx-servers
>>> Applying project ➡ nginx-servers 🚀 
>>> [nginx1] - anise repo update && anise upgrade -y - ☕ 
ℹ️  Repository:              geaaru-repo-index is already up to date.
ℹ️  Repository:               macaroni-phoenix is already up to date.
ℹ️  Repository:       macaroni-phoenix-testing is already up to date.
ℹ️  Repository:               mottainai-stable is already up to date.
ℹ️  Repository:               macaroni-commons is already up to date.
ℹ️  Repository:       macaroni-commons-testing is already up to date.

...

>>> [nginx1] Syncing 1 resources... - 🚌 
>>> [nginx1] - [ 1/ 1] /etc/nginx//nginx.conf - ✔ 
>>> [nginx1] - ubuntu=$(cat /etc/os-release | grep ID| grep ubuntu | wc -l); if [ ${ubuntu} == 1 ] ; then systemctl restart nginx ; else /etc/init.d/nginx restart ; fi - ☕ 
 * Checking nginx' configuration ... [ ok ]
 * Stopping nginx ... [ ok ]
 * Starting nginx ... [ ok ]
🎉 All done!
```

Enjoy!


## Available commands

### Add a new remote

```bash
$> ssh-compose remote a test --host 172.18.10.192 --user geaaru --auth-method password --pass pass

$> ssh-compose remote a mynode2 --host 10.20.20.10 --user geaaru --auth-method publickey \
        --privatekey-file ~/.ssh/id_dsa  --default
```

### Remove a remote

```bash
$> ssh-compose remote rm mynode1
```

### Show the remotes list

```bash
$> ssh-compose remote list
|      NAME      |          URL          | AUTHMETHOD |  USER  |
|----------------|-----------------------|------------|--------|
| mynode1        | tcp::10.10.10.10:22   | password   | geaaru |
| mynode2        | tcp::10.20.20.10:22   | publickey  | geaaru |
| test (default) | tcp::172.18.10.192:22 | password   | root   |

```

### Deploy an environment

```bash

$> ssh-compose apply myproject

# Disable hooks with flag foo
$> ssh-compose apply --disable-flag foo

# Execute only hooks with flag foo
$> ssh-compose apply --enable-flag foo

```

A stupid example of a project is [here](https://raw.githubusercontent.com/MottainaiCI/ssh-compose/master/contrib/envs/example.yaml).

Hereinafter, an example of the *apply* output:

```bash
$> ssh-compose a ssh-compose-helloword
Apply project ssh-compose-helloword
>>> [node-example] - echo "Welcome to ssh-compose-${SSH_COMPOSE_VERSION}!!!" - ☕ 
Welcome to ssh-compose-0.0.1!!!
>>> [node-example] - nodestr=$(echo "${SSH_COMPOSE_PROJECT}" | jq .node -r) ; echo "Node name $(echo "${nodestr}" | jq .name -r)" - ☕ 
Node name node-example
>>> [node-example] - echo "Variable var1 equal to $(echo "${SSH_COMPOSE_PROJECT}" | jq .var1 -r)" - ☕ 
Variable var1 equal to value1
All done.

```

### Enter in the shell of the remote configured

```bash

$> ssh-compose shell node1

```

### Push files/directory to a remote node

```bash

$> ssh-compose push <remote> ./local-path/ /remote-path/

```

### Pull files/directory from a remote node to a local path

```bash

$> ssh-compose pull <remote> ./remote-path/ /local-path/

```

### Show list of the project

```bash

$> ssh-compose project list

```

## Community

The ssh-compose devs team is available through the [Mottainai](https://join.slack.com/t/mottainaici/shared_invite/zt-zdmrc651-IvxE9j~TT5ssv_CVo51uZg) Slack channel.
