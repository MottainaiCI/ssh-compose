# SSH Compose

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
