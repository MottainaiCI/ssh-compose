# SSH Compose

**ssh-compose** is the system of the **lxd-compose** project and supplies a way to deploy a complex environment
using SSH protocol.

It permits to organize and trace all configuration steps of infrastructure and create test suites.

All configuration files could be created at runtime through two different template engines: Mottainai or Jinja2 (require `j2cli` tool).

It's under heavy development phase and specification could be changed in the near future.

## Documentation

Incoming...

## Getting Started

### Deploy an environment

```bash

$> ssh-compose apply myproject

# Disable hooks with flag foo
$> ssh-compose apply --disable-flag foo

# Execute only hooks with flag foo
$> ssh-compose apply --enable-flag foo

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
