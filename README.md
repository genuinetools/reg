# reg

[![Travis CI](https://travis-ci.org/jessfraz/reg.svg?branch=master)](https://travis-ci.org/jessfraz/reg)

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
   @jessfraz <no-reply@butts.com>

COMMANDS:
     delete, rm       delete a specific reference of a repository
     list, ls         list all repositories
     manifest         get the json manifest for the specific reference of a repository
     vulns            get a vulnerability report for the image from CoreOS Clair
     tags             get the tags for a repository
     download, layer  download a layer for the specific reference of a repository
     help, h          Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --debug, -d                 run in debug mode
   --username value, -u value  username for the registry
   --password value, -p value  password for the registry
   --registry value, -r value  URL to the provate registry (ex. r.j3ss.co)
   --help, -h                  show help
   --version, -v               print the version
```

**Get a vulnerability report**

```console
$ $ reg vulns --clair https://clair.j3ss.co chrome
Found 32 vulnerabilities
CVE-2015-5180: [Low]

https://security-tracker.debian.org/tracker/CVE-2015-5180
-----------------------------------------
CVE-2016-9401: [Low]
popd in bash might allow local users to bypass the restricted shell and cause a use-after-free via a crafted address.
https://security-tracker.debian.org/tracker/CVE-2016-9401
-----------------------------------------
CVE-2016-3189: [Low]
Use-after-free vulnerability in bzip2recover in bzip2 1.0.6 allows remote attackers to cause a denial of service (crash) via a crafted bzip2 file, related to block ends set to before the start of the block.
https://security-tracker.debian.org/tracker/CVE-2016-3189
-----------------------------------------
CVE-2011-3389: [Medium]
The SSL protocol, as used in certain configurations in Microsoft Windows and Microsoft Internet Explorer, Mozilla Firefox, Google Chrome, Opera, and other products, encrypts data by using CBC mode with chained initialization vectors, which allows man-in-the-middle attackers to obtain plaintext HTTP headers via a blockwise chosen-boundary attack (BCBA) on an HTTPS session, in conjunction with JavaScript code that uses (1) the HTML5 WebSocket API, (2) the Java URLConnection API, or (3) the Silverlight WebClient API, aka a "BEAST" attack.
https://security-tracker.debian.org/tracker/CVE-2011-3389
-----------------------------------------
CVE-2016-5318: [Medium]
Stack-based buffer overflow in the _TIFFVGetField function in libtiff 4.0.6 and earlier allows remote attackers to crash the application via a crafted tiff.
https://security-tracker.debian.org/tracker/CVE-2016-5318
-----------------------------------------
CVE-2016-9318: [Medium]
libxml2 2.9.4 and earlier, as used in XMLSec 1.2.23 and earlier and other products, does not offer a flag directly indicating that the current document may be read but other files may not be opened, which makes it easier for remote attackers to conduct XML External Entity (XXE) attacks via a crafted document.
https://security-tracker.debian.org/tracker/CVE-2016-9318
-----------------------------------------
CVE-2015-7554: [High]
The _TIFFVGetField function in tif_dir.c in libtiff 4.0.6 allows attackers to cause a denial of service (invalid memory write and crash) or possibly have unspecified other impact via crafted field data in an extension tag in a TIFF image.
https://security-tracker.debian.org/tracker/CVE-2015-7554
-----------------------------------------
Unknown: 2
Negligible: 23
Low: 3
Medium: 3
High: 1
```

**Known Issues**

`reg` does not work to:
* output image history
