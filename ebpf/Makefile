ARCH := x86
BUILD_DIR := builds
TARGET_PREFIX := bpf-
TARGET_SUFFIX := -amd64
UNAME_ARCH := $(shell uname -m)

ifeq ($(UNAME_ARCH),aarch64)
	ARCH := arm64
	TARGET_SUFFIX := -arm64
endif

PROJECTS = $(shell find * -maxdepth 1 -type d | grep -v includes | grep -v builds)

LIBBPF_OBJ := /usr/lib/$(UNAME_ARCH)-linux-gnu/libbpf.a

.PHONY: all $(PROJECTS)

all: $(PROJECTS)

go_env := CC=clang CGO_CFLAGS="-I /usr/include/$(UNAME_ARCH)-linux-gnu" CGO_LDFLAGS="$(LIBBPF_OBJ)"

$(PROJECTS):
	@echo "Start build the" \"$@\" "project"
	$(eval TARGET := $(TARGET_PREFIX)$@$(TARGET_SUFFIX))
	$(eval TARGET_PATH := $(BUILD_DIR)/$(TARGET))
	@echo "Compiling" \"$(TARGET)\" "GO program in progress..."
	${go_env} go build -o ./$(TARGET_PATH) $@/main.go
	@echo "Successfully compiled" \"$(TARGET)\" "GO program! Location: "$(TARGET_PATH)
	$(eval TARGET_BPF := $(TARGET).bpf.o)
	@echo "Compiling" \"$(TARGET_BPF)\" "eBPF program in progress..."
	clang \
		-D __TARGET_ARCH_$(ARCH) \
		-I /usr/include/$(UNAME_ARCH)-linux-gnu \
		-O2 -c -target bpf \
		-g \
		-o $(BUILD_DIR)/$(TARGET_BPF) $@/*.bpf.c
	@echo "Successfully compiled" \"$(TARGET_BPF)\" "eBPF program! Location:" $(BUILD_DIR)/$(TARGET_BPF)

.PHONY: clean
clean:
	go clean
	rm -r builds/

