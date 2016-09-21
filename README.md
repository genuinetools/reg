# reg

[![Travis CI](https://travis-ci.org/jfrazelle/reg.svg?branch=master)](https://travis-ci.org/jfrazelle/reg)

Docker registry v2 command line client.

> **NOTE:** There is a way better, _maintained_ version of this @
> [`docker-ls`](https://github.com/mayflower/docker-ls)

**Auth**

`reg` will automatically try to parse your docker config credentials, but if
not present, you can pass through flags directly.

**List Repositories and Tags**

```console
# this command might take a while if you have hundreds of images like I do
$ reg -r r.j3ss.co ls
Repositories for r.j3ss.co
REPO                  TAGS
ab                    latest
android-tools         latest
apt-file              latest
atom                  latest
audacity              latest
awscli                latest
beeswithmachineguns   latest
buttslock             latest
camlistore            latest
cathode               latest
cf-reset-cache        latest
cheese                latest
chrome                beta, latest, stable
...
```

**Usage**

```console
$ reg
NAME:
   reg - Docker registry v2 client.

USAGE:
   reg [global options] command [command options] [arguments...]

VERSION:
   v0.2.0

AUTHOR(S):
   @jfrazelle <no-reply@butts.com>

COMMANDS:
     delete    delete a specific reference of a repository
     list, ls  list all repositories
     manifest  get the json manifest for the specific reference of a repository
     tags      get the tags for a repository
     help, h   Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --debug, -d                 run in debug mode
   --username value, -u value  username for the registry
   --password value, -p value  password for the registry
   --registry value, -r value  URL to the provate registry (ex. r.j3ss.co)
   --help, -h                  show help
   --version, -v               print the version
```

**Known Issues**

`reg` does not work to:
* output image history
