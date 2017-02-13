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
CVE-2016-2781: [Unknown]
chroot in GNU coreutils, when used with --userspec, allows local users to escape to the parent session via a crafted TIOCSTI ioctl call, which pushes characters to the terminal's input buffer.
https://security-tracker.debian.org/tracker/CVE-2016-2781
-----------------------------------------
CVE-2016-10095: [Unknown]

https://security-tracker.debian.org/tracker/CVE-2016-10095
-----------------------------------------
CVE-2007-5686: [Negligible]
initscripts in rPath Linux 1 sets insecure permissions for the /var/log/btmp file, which allows local users to obtain sensitive information regarding authentication attempts.  NOTE: because sshd detects the insecure permissions and does not log certain events, this also prevents sshd from logging failed authentication attempts by remote attackers.
https://security-tracker.debian.org/tracker/CVE-2007-5686
-----------------------------------------
CVE-2016-6251: [Negligible]

https://security-tracker.debian.org/tracker/CVE-2016-6251
-----------------------------------------
CVE-2013-4235: [Negligible]

https://security-tracker.debian.org/tracker/CVE-2013-4235
-----------------------------------------
CVE-2005-2541: [Negligible]
Tar 1.15.1 does not properly warn the user when extracting setuid or setgid files, which may allow local users or remote attackers to gain privileges.
https://security-tracker.debian.org/tracker/CVE-2005-2541
-----------------------------------------
CVE-2010-4756: [Negligible]
The glob implementation in the GNU C Library (aka glibc or libc6) allows remote authenticated users to cause a denial of service (CPU and memory consumption) via crafted glob expressions that do not match any pathnames, as demonstrated by glob expressions in STAT commands to an FTP daemon, a different vulnerability than CVE-2010-2632.
https://security-tracker.debian.org/tracker/CVE-2010-4756
-----------------------------------------
CVE-2010-4051: [Negligible]
The regcomp implementation in the GNU C Library (aka glibc or libc6) through 2.11.3, and 2.12.x through 2.12.2, allows context-dependent attackers to cause a denial of service (application crash) via a regular expression containing adjacent bounded repetitions that bypass the intended RE_DUP_MAX limitation, as demonstrated by a {10,}{10,}{10,}{10,}{10,} sequence in the proftpd.gnu.c exploit for ProFTPD, related to a "RE_DUP_MAX overflow."
https://security-tracker.debian.org/tracker/CVE-2010-4051
-----------------------------------------
CVE-2010-4052: [Negligible]
Stack consumption vulnerability in the regcomp implementation in the GNU C Library (aka glibc or libc6) through 2.11.3, and 2.12.x through 2.12.2, allows context-dependent attackers to cause a denial of service (resource exhaustion) via a regular expression containing adjacent repetition operators, as demonstrated by a {10,}{10,}{10,}{10,} sequence in the proftpd.gnu.c exploit for ProFTPD.
https://security-tracker.debian.org/tracker/CVE-2010-4052
-----------------------------------------
CVE-2017-5932: [Negligible]

https://security-tracker.debian.org/tracker/CVE-2017-5932
-----------------------------------------
CVE-2011-3374: [Negligible]

https://security-tracker.debian.org/tracker/CVE-2011-3374
-----------------------------------------
CVE-2013-0340: [Negligible]
expat 2.1.0 and earlier does not properly handle entities expansion unless an application developer uses the XML_SetEntityDeclHandler function, which allows remote attackers to cause a denial of service (resource consumption), send HTTP requests to intranet servers, or read arbitrary files via a crafted XML document, aka an XML External Entity (XXE) issue.  NOTE: it could be argued that because expat already provides the ability to disable external entity expansion, the responsibility for resolving this issue lies with application developers; according to this argument, this entry should be REJECTed, and each affected application would need its own CVE.
https://security-tracker.debian.org/tracker/CVE-2013-0340
-----------------------------------------
CVE-2007-6755: [Negligible]
The NIST SP 800-90A default statement of the Dual Elliptic Curve Deterministic Random Bit Generation (Dual_EC_DRBG) algorithm contains point Q constants with a possible relationship to certain "skeleton key" values, which might allow context-dependent attackers to defeat cryptographic protection mechanisms by leveraging knowledge of those values.  NOTE: this is a preliminary CVE for Dual_EC_DRBG; future research may provide additional details about point Q and associated attacks, and could potentially lead to a RECAST or REJECT of this CVE.
https://security-tracker.debian.org/tracker/CVE-2007-6755
-----------------------------------------
CVE-2010-0928: [Negligible]
OpenSSL 0.9.8i on the Gaisler Research LEON3 SoC on the Xilinx Virtex-II Pro FPGA uses a Fixed Width Exponentiation (FWE) algorithm for certain signature calculations, and does not verify the signature before providing it to a caller, which makes it easier for physically proximate attackers to determine the private key via a modified supply voltage for the microprocessor, related to a "fault-based attack."
https://security-tracker.debian.org/tracker/CVE-2010-0928
-----------------------------------------
CVE-2012-3878: [Negligible]

https://security-tracker.debian.org/tracker/CVE-2012-3878
-----------------------------------------
CVE-2011-4116: [Negligible]

https://security-tracker.debian.org/tracker/CVE-2011-4116
-----------------------------------------
CVE-2017-5563: [Negligible]
LibTIFF version 4.0.7 is vulnerable to a heap-based buffer over-read in tif_lzw.c resulting in DoS or code execution via a crafted bmp image to tools/bmp2tiff.
https://security-tracker.debian.org/tracker/CVE-2017-5563
-----------------------------------------
CVE-2014-8130: [Negligible]

https://security-tracker.debian.org/tracker/CVE-2014-8130
-----------------------------------------
CVE-2013-4392: [Negligible]
systemd, when updating file permissions, allows local users to change the permissions and SELinux security contexts for arbitrary files via a symlink attack on unspecified files.
https://security-tracker.debian.org/tracker/CVE-2013-4392
-----------------------------------------
CVE-2012-0039: [Negligible]
** DISPUTED ** GLib 2.31.8 and earlier, when the g_str_hash function is used, computes hash values without restricting the ability to trigger hash collisions predictably, which allows context-dependent attackers to cause a denial of service (CPU consumption) via crafted input to an application that maintains a hash table.  NOTE: this issue may be disputed by the vendor; the existence of the g_str_hash function is not a vulnerability in the library, because callers of g_hash_table_new and g_hash_table_new_full can specify an arbitrary hash function that is appropriate for the application.
https://security-tracker.debian.org/tracker/CVE-2012-0039
-----------------------------------------
CVE-2015-3276: [Negligible]
The nss_parse_ciphers function in libraries/libldap/tls_m.c in OpenLDAP does not properly parse OpenSSL-style multi-keyword mode cipher strings, which might cause a weaker than intended cipher to be used and allow remote attackers to have unspecified impact via unknown vectors.
https://security-tracker.debian.org/tracker/CVE-2015-3276
-----------------------------------------
CVE-2014-8166: [Negligible]

https://security-tracker.debian.org/tracker/CVE-2014-8166
-----------------------------------------
CVE-2004-0971: [Negligible]
The krb5-send-pr script in the kerberos5 (krb5) package in Trustix Secure Linux 1.5 through 2.1, and possibly other operating systems, allows local users to overwrite files via a symlink attack on temporary files.
https://security-tracker.debian.org/tracker/CVE-2004-0971
-----------------------------------------
CVE-2011-3374: [Negligible]

https://security-tracker.debian.org/tracker/CVE-2011-3374
-----------------------------------------
CVE-2016-2779: [Negligible]
runuser in util-linux allows local users to escape to the parent session via a crafted TIOCSTI ioctl call, which pushes characters to the terminal's input buffer.
https://security-tracker.debian.org/tracker/CVE-2016-2779
-----------------------------------------
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
