UNAME_ARCH=$(shell uname -m)
ARCH := x86
TARGET := bpf-diskfailure-amd64
ifeq ($(UNAME_ARCH),aarch64)
	ARCH=arm64
	TARGET=bpf-diskfailure-arm64
endif

TARGET_BPF := $(TARGET).bpf.o

GO_SRC := *.go
BPF_SRC := *.bpf.c

LIBBPF_HEADERS := /usr/include/bpf
LIBBPF_OBJ := /usr/lib/$(UNAME_ARCH)-linux-gnu/libbpf.a

.PHONY: all
all: $(TARGET) $(TARGET_BPF)

go_env := CC=clang CGO_CFLAGS="-I /usr/include/$(UNAME_ARCH)-linux-gnu" CGO_LDFLAGS="$(LIBBPF_OBJ)"
$(TARGET): $(GO_SRC)
	$(go_env) go build -o $(TARGET)

$(TARGET_BPF): $(BPF_SRC)
	clang \
		-D __TARGET_ARCH_$(ARCH) \
		-I /usr/include/$(UNAME_ARCH)-linux-gnu \
		-O2 -c -target bpf \
		-g \
		-o $@ $<

.PHONY: clean
clean:
	go clean

