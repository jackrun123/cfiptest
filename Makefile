# 设置编译器和链接器的一些参数

LDFLAGS := -ldflags="-s -w"

# 默认的目标
all: build_linux_arm64

build_linux_arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o cfiptest_arm64 .

.PHONY: all build_linux_arm64