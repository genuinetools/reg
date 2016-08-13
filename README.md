# reg

[![Travis CI](https://travis-ci.org/jfrazelle/reg.svg?branch=master)](https://travis-ci.org/jfrazelle/reg)

Docker registry v2 client.

> **NOTE:** There is a way better version of this @  [`docker-ls`](https://github.com/mayflower/docker-ls)

**Auth**

`reg` will automatically try to parse your docker config credentials, but if
not saved there you can pass through flags directly.

**List Repositories and Tags**

```console
$ ./reg
Repositories for registry.jess.co
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
$ reg --help
 _ __ ___  __ _
| '__/ _ \/ _` |
| | |  __/ (_| |
|_|  \___|\__, |
          |___/

 Docker registry v2 client.
 Version: v0.1.0

  -d	run in debug mode
  -p string
    	Password for the registry
  -r string
    	Url to the private registry (ex. https://registry.jess.co)
  -u string
    	Username for the registry
  -v	print version and exit (shorthand)
  -version
    	print version and exit
```

**Known Issues**

`reg` does not work with
* unauthenticated registries
* http basic auth
* output image history
 
For more advanced registry usage please use [`docker-ls`](https://github.com/mayflower/docker-ls)
