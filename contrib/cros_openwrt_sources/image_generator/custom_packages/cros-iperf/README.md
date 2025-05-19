This package contains a static version of IPerf2 whose version is compatible
with the one contained in the ChromeOS test image:
https://source.chromium.org/chromiumos/chromiumos/codesearch/+/main:src/third_party/portage-stable/net-misc/iperf/.

This is done to overcome several limitations of the default OpenWRT IPerf
package (b/312265689 ):
1. The two versions are not compatible for bidirectional UDP tests and the
   server->client connection is always refused.
2. The default distribution does not accurately detect threading support
   during auto configuration and fails to enable the HAVE_THREAD/HAVE_POSIX_THREAD
   directives. This prevents UDP tests from sending the server report back to
   the client at the end of the test.
