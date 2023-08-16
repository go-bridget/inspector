# inspector

This is a very basic ssh helper tool to manage a smaller (few 100s up to
a few 1000s) fleet of servers. The main point of inspector is to provide
key insights into system details, for example, so you know which software
you're running, OS and kernel versions, which hosts need upgrades,
performance metrics, and basically whatever you can script into a ssh
command.

To configure inspector, create `inspector.yml` with:

- The `aliases` section gives you the ability to create a
shorthand commands, which you can invoke over with the first argument.
Using `run` is a reserved keyword.
- The `columns` section defines output columns for each server. When you
run inspector without arguments, all these values are being retrieved and
a table is printed.
- The `servers` section is a list of remote servers to connect to.

Example configuration:

~~~yaml
aliases:
  uptime: uptime
  kernel: uname -v

columns:
- name: Docker
  command: docker version -f '{{ .Server.Version }}' 2>/dev/null || echo None
- name: Containers
  command: docker ps -a --format '{{ .Names }}' | wc -l
- name: Go
  command: go version 2>/dev/null || echo None

servers:
- docker1
- docker3
- docker4
- docker5
- docker6
- docker7
- docker8
- docker9
~~~

Asuming `inspector` is in your execution path, you can then run:

- `inspector` - Provides a general overview of your defined servers (columns)
- `inspector uptime` - Runs the `uptime` alias over your servers
- `inspector run uname -r` - Runs `uname -r` on all hosts

Example output:

~~~text
Server               Docker    Containers  Go                               
docker1              19.03.5   3           None                             
docker3              18.09.0   4           None                             
docker4              18.09.0   4           None                             
docker5              19.03.11  3           None                             
docker6              18.09.5   14          None                             
docker7              18.09.5   18          None                             
docker8              18.09.5   18          None                             
docker9              18.09.5   19          None    
~~~

## Performance

Each ssh connection is created in parallel and individual columns are run
serially. This means that the response from your fleet will be available
to you within seconds, not minutes.

For a fleet of 60 servers, getting the complete info or uptime, just like
defined above, the response takes 2 seconds. If the limitation is the
connection rate itself, then we can asume that we can query 1000 servers
and get the complete response in about 30 seconds.

- 10 servers about 0.6sec
- 60 servers about 1.8sec
- 1000 severs about 20-25 sec? (estimate)

> If you have a fleet of 1000s of servers, I'd be interested to know
> how well inspector performs for running `uptime` on all of them.

## Other

The authentication uses SSH Agent, or a PrivateKey which should either be
under `.ssh/id_rsa` or `$HOME/.ssh/id_rsa`. The `root` user is used to
connect to remote hosts.

## Ideas

Depending on our internal usage of inspector, the following features may
be added. If you're using inspector and are familiar with Go, feel free
to open an issue to discuss requirements before submitting a PR.

- [ ] Ability to sort by column
- [ ] Set non-root user and use sudo
- [x] Output machine readable results (use `--json`)
- [ ] Enable support for `known_hosts`
- [ ] Daemon mode with continous monitoring + prometheus export?
- [ ] A better commands and flags implementation (* Don't bother with this one, we have some ideas and are very particular / peculiar about flag packages)

