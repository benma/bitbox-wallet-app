include ../../android-env.mk.inc
include ../../version.mk.inc

override GO_TAGS := timetzdata sqlcipher

# Set -glflags to fix the vendor issue with gomobile, see: https://github.com/golang/go/issues/67927#issuecomment-2241523694
build-android:
	# androidapi version should match minSdkVersion in frontends/android/BitBoxApp/app/build.gradle
	ANDROID_HOME=${ANDROID_SDK_ROOT} gomobile bind -x -a -glflags="-mod=readonly" -tags="$(GO_TAGS)" -ldflags="-s -w $(GO_VERSION_LDFLAGS)" -trimpath -target android -androidapi 24 .
build-ios:
	gomobile bind -x -a -glflags="-mod=readonly" -tags="$(GO_TAGS)" -trimpath -ldflags="-s -w $(GO_VERSION_LDFLAGS)" -target ios,iossimulator .
clean:
	rm -f mobileserver.aar mobileserver-sources.jar
