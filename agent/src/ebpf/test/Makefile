ifeq ($(V),1)
        Q =
        msg =
else
        Q = @
        msg = @printf '  %-8s %s%s\n' "$(1)" "$(notdir $(2))" "$(if $(3), $(3))";
endif

CC ?= gcc

CFLAGS ?= -std=gnu99 --static -g -O2 -ffunction-sections -fdata-sections -fPIC -fno-omit-frame-pointer -m64 -Wall -Wno-sign-compare -Wno-unused-parameter -Wno-missing-field-initializers
CFLAGS += -I../libbpf/src -I../libbpf/include -I../libbpf/include/uapi -I../

EXECS := test_symbol test_offset
LDLIBS += ../libebpf.a -lbddisasm -ldwarf -lelf -lz -lpthread

all: $(EXECS) 
% : %.c
	$(call msg,TEST,$@)
	$(Q)$(CC) $(CFLAGS) -o $@ $^ $(LDLIBS)
	$(Q)./$@

clean:
	$(Q)rm -rf $(EXECS)
	
.PHONY: all clean
