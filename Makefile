# GoAnime — Build Targets
# Requires: Go 1.26+, gomobile, Android SDK + NDK

ANDROID_API := 26
AAR_OUTPUT := android/app/libs/gobridge.aar
GOBRIDGE_PKG := ./gobridge/

.PHONY: help gobridge-test android-lib android-app clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

gobridge-test: ## Run Go bridge unit tests
	go test -short -v $(GOBRIDGE_PKG)

gobridge-integration: ## Run Go bridge integration tests (requires network)
	go test -v $(GOBRIDGE_PKG)

android-lib: gobridge-test ## Compile Go bridge to Android .aar
	@mkdir -p android/app/libs
	gomobile bind -target=android -androidapi $(ANDROID_API) -o $(AAR_OUTPUT) $(GOBRIDGE_PKG)
	@echo "Built: $(AAR_OUTPUT)"

android-app: android-lib ## Build Android APK (debug)
	cd android && ./gradlew assembleDebug

android-release: android-lib ## Build Android APK (release)
	cd android && ./gradlew assembleRelease

clean: ## Remove build artifacts
	rm -f $(AAR_OUTPUT)
	rm -rf android/app/build
	rm -rf bin/

# Development helpers
fmt: ## Format all Go code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

test: ## Run all Go tests (short mode)
	go test -short ./...
