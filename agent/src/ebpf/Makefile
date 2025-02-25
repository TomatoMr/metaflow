ifeq ($(V),1)
	Q =
	msg =
else
	Q = @
	msg = @printf '  %-8s %s%s\n' "$(1)" "$(notdir $(2))" "$(if $(3), $(3))";
endif

CC ?= gcc
OBJDIR ?= .
STATIC_OBJDIR := $(OBJDIR)/staticobjs

define compile_socket_trace_elf
	@echo "  COMPILE ELF kernel version $(1)"
	@cd kernel && make clean --no-print-directory && make socket_trace.elf $(2) --no-print-directory && cd ../
	@echo "  Generate file user/socket_trace_bpf_$(strip $1).c"
	@./tools/ebpftobuffer kernel/socket_trace.elf user/socket_trace_bpf_$(strip $1).c socket_trace_$(strip $1)_ebpf_data
endef

CURRDIR := $(PWD)
CLANG_VER := $(shell /usr/bin/clang --version | grep "version" | awk '{print $$3}')
CLANG_VER_MAIN := $(shell /usr/bin/clang --version | grep "version" | awk '{print $$3}' | awk -F. '{print $$1}' | awk '{print int($$0)}')
LIBTRACE := libebpf.a
LIBBPF_DIR = libbpf/src
OBJS := $(LIBBPF_DIR)/bpf.o \
	$(LIBBPF_DIR)/btf.o \
	$(LIBBPF_DIR)/libbpf.o \
	$(LIBBPF_DIR)/libbpf_errno.o \
	$(LIBBPF_DIR)/netlink.o \
	$(LIBBPF_DIR)/nlattr.o \
	$(LIBBPF_DIR)/str_error.o \
	$(LIBBPF_DIR)/libbpf_probes.o \
	$(LIBBPF_DIR)/bpf_prog_linfo.o \
	$(LIBBPF_DIR)/xsk.o \
	$(LIBBPF_DIR)/btf_dump.o \
	$(LIBBPF_DIR)/hashmap.o \
	$(LIBBPF_DIR)/ringbuf.o \
	$(LIBBPF_DIR)/strset.o \
	$(LIBBPF_DIR)/linker.o \
	$(LIBBPF_DIR)/gen_loader.o \
	$(LIBBPF_DIR)/relo_core.o \
	user/bcc/bcc_elf.o \
	user/bcc/bcc_proc.o \
	user/bcc/libbpf.o \
	user/bcc/perf_reader.o \
	user/common.o \
	user/probe.o \
	user/symbol.o \
	user/go_tracer.o \
	user/table.o \
	user/ring.o \
	user/tracer.o \
	user/socket.o \
	user/log.o \
	user/ctrl.o \
	user/offset.o

STATIC_OBJS := $(addprefix $(STATIC_OBJDIR)/,$(OBJS))

CFLAGS ?= -std=gnu99 -g -O2 -ffunction-sections -fdata-sections -fPIC -fno-omit-frame-pointer -m64 -Wall -Wno-sign-compare -Wno-unused-parameter -Wno-missing-field-initializers
CFLAGS += $(MACHINE_CFLAGS) -D_LARGEFILE64_SOURCE -D_FILE_OFFSET_BITS=64 -I$(LIBBPF_DIR) -Ilibbpf/include -Ilibbpf/include/uapi -I.
LDFLAGS ?= -lbddisasm -ldwarf -lelf -lz -lpthread

all: build

ELFFILES = ./.data_done
$(ELFFILES):
	$(call msg,Clang/LLVM,,$(CLANG_VER))
	@if [ $(CLANG_VER_MAIN) -lt 10 ]; then \
                echo "  check llvm-clang fail. expect Clang/LLVM 10+" && exit 1; \
        fi
	@rm -rf data
	$(call msg,Tools,,tools/ebpftobuffer)
	@gcc tools/ebpftobuffer.c -o tools/ebpftobuffer
	$(call compile_socket_trace_elf, common)
	$(call compile_socket_trace_elf, 5_2, LINUX_VER_5_2=1)
	$(call compile_socket_trace_elf, core, CORE=1)
	@touch $(ELFFILES)

$(STATIC_OBJDIR) $(SHARED_OBJDIR):
	$(call msg,MKDIR,$@)
	$(Q)mkdir -p $@/$(LIBBPF_DIR)
	$(Q)mkdir -p $@/user/bcc

$(STATIC_OBJDIR)/%.o: %.c | $(STATIC_OBJDIR)
	$(call msg,CC,$@)
	$(Q)$(CC) $(CFLAGS) -c $< -o $@

$(LIBTRACE): $(STATIC_OBJS)
	$(call msg,AR,$@)
	$(Q)$(AR) rcs $@ $^

$(LIBBPF_DIR):
	@cd ../../../ && git submodule update --init --recursive && cd $(CURRDIR)
	
build: $(LIBBPF_DIR) $(ELFFILES) $(LIBTRACE)

tools: $(LIBBPF_DIR) $(LIBTRACE)
	$(call msg,TOOLS,metaflow-ebpfctl)
	$(Q)$(CC) $(CFLAGS) --static -g -O2 user/ctrl_tracer.c user/ctrl.c libebpf.a -o metaflow-ebpfctl -lelf -lz -lpthread

rust-sample: $(LIBBPF_DIR) $(ELFFILES) $(LIBTRACE)
	$(call msg,Current-DIR,,$(CURR))
	$(Q)rm -rf samples/rust/src/ebpf.rs
	$(Q)cp mod.rs samples/rust/src/ebpf.rs
	$(Q)rm -rf samples/rust/target
	$(Q)cd samples/rust/ && cargo build --release && cd ../../
	$(call msg,RUST-SAMPLE,,samples/rust/target/release/rust_sample)

clean:
	$(Q)$(MAKE) -C kernel clean --no-print-directory
	$(Q)$(MAKE) -C test clean --no-print-directory
	$(Q)rm -rf $(ELFFILES) data metaflow-ebpfctl $(STATIC_OBJDIR) $(LIBTRACE) *.a
	$(Q)rm -rf samples/rust/target

test: $(LIBBPF_DIR) $(ELFFILES) $(LIBTRACE)
	$(Q)$(MAKE) -C test --no-print-directory

.PHONY: all build clean tools test

