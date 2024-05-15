# SSH Compose

**ssh-compose** is the sister of the **lxd-compose** project and supplies a way to deploy a complex environment
using SSH protocol.

It permits to organize and trace all configuration steps of infrastructure and create test suites.

All configuration files could be created at runtime through two different template engines: Mottainai or Jinja2 (require `j2cli` tool).

It's under heavy development phase and specification could be changed in the near future.

## Documentation

Incoming...

## TODO for the first release

- [ ] Integrate SFTP sync
- [ ] Complete support of Private Key with Password
- [ ] Cleanup logging
- [ ] CD/CI integration

## Getting Started

The file `.ssh-compose.yml` contains the rules about `ssh-compose` reads and loads the environments and projects.

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
AcceptEnv SSH_COMPOSE SSH_COMPOSE_*

```

### Add a new remote

```bash
$> ssh-compose remote a test --host 172.18.10.192 --user geaaru --auth-method password mynode --pass pass

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

### Show list of the project

```bash

$> ssh-compose project list

```

## Community

The ssh-compose devs team is available through the [Mottainai](https://join.slack.com/t/mottainaici/shared_invite/zt-zdmrc651-IvxE9j~TT5ssv_CVo51uZg) Slack channel.
