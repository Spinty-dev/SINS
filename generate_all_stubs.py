import subprocess
import os
import re

def get_local_symbols():
    print("Extracting symbols from local Arch Linux libsystemd...")
    cmd = "nm -D /usr/lib/libsystemd.so.0 | grep ' T ' | sed 's/.* T //'"
    result = subprocess.check_output(cmd, shell=True).decode()
    return [line.strip() for line in result.split('\n') if line.strip()]

def main():
    raw_symbols = get_local_symbols()
    vgroups = {}
    
    for s in raw_symbols:
        if '@@' in s:
            name, ver_str = s.split('@@')
        elif '@' in s:
            name, ver_str = s.split('@')
        else:
            name = s
            ver_str = "LIBSYSTEMD_209"
            
        vnum_match = re.search(r'LIBSYSTEMD_(\d+)', ver_str)
        vnum = int(vnum_match.group(1)) if vnum_match else 209
            
        if vnum not in vgroups: vgroups[vnum] = []
        if name not in vgroups[vnum]: vgroups[vnum].append(name)

    all_names = sorted(set([n for names in vgroups.values() for n in names]))
    print(f"Total symbols to wrap: {len(all_names)}")

    with open("pkg/libsystemd/stub.c", "w") as f:
        f.write("#define _GNU_SOURCE\n#include <stdio.h>\n#include <stdint.h>\n#include <errno.h>\n#include <stdlib.h>\n#include <string.h>\n#include <dlfcn.h>\n#include <unistd.h>\n#include <stdarg.h>\n#include <time.h>\n#include <sys/uio.h>\n\n")
        f.write("static void* real_lib = NULL;\n")
        f.write("__attribute__((constructor)) static void init_lib() { real_lib = dlopen(\"libelogind.so.0\", RTLD_LAZY); }\n\n")
        f.write("static void* get_fn(const char* name) { if (!real_lib) init_lib(); return dlsym(real_lib, name); }\n\n")
        
        # JOURNAL WRITER
        f.write("static void sins_journal_log(const char* msg) {\n")
        f.write("    FILE* f = fopen(\"/tmp/sins-journal.log\", \"a\");\n")
        f.write("    if (!f) return;\n")
        f.write("    time_t now = time(NULL);\n")
        f.write("    char* ts = ctime(&now);\n")
        f.write("    if (ts) ts[strlen(ts)-1] = 0;\n")
        f.write("    fprintf(f, \"[%s] %s\\n\", ts ? ts : \"?\", msg);\n")
        f.write("    fclose(f);\n")
        f.write("}\n\n")

        # sd-journal UNIVERSAL MOCKS (Enhanced)
        f.write("int sd_journal_print(int priority, const char *format, ...) {\n")
        f.write("    char buf[1024]; va_list aq; va_start(aq, format); vsnprintf(buf, sizeof(buf), format, aq); va_end(aq);\n")
        f.write("    sins_journal_log(buf); return 0;\n}\n")
        
        f.write("int sd_journal_print_with_location(int priority, const char *file, const char *line, const char *func, const char *format, ...) {\n")
        f.write("    char buf[1024]; va_list aq; va_start(aq, format); vsnprintf(buf, sizeof(buf), format, aq); va_end(aq);\n")
        f.write("    sins_journal_log(buf); return 0;\n}\n")

        f.write("int sd_journal_send(const char *format, ...) {\n")
        f.write("    sins_journal_log(format); return 0;\n}\n")

        f.write("int sd_journal_sendv(const struct iovec *iov, int n) {\n")
        f.write("    for(int i=0; i<n; i++) { if (iov[i].iov_base) sins_journal_log((char*)iov[i].iov_base); }\n")
        f.write("    return 0;\n}\n")

        f.write("int sd_journal_sendv_with_location(const struct iovec *iov, int n, const char *file, const char *line, const char *func) {\n")
        f.write("    return sd_journal_sendv(iov, n);\n}\n")

        journal_symbols = [
            "sd_journal_add_conjunction", "sd_journal_add_disjunction", "sd_journal_add_match", "sd_journal_close",
            "sd_journal_enumerate_available_data", "sd_journal_enumerate_available_unique", "sd_journal_enumerate_data",
            "sd_journal_enumerate_fields", "sd_journal_enumerate_unique", "sd_journal_flush_matches", "sd_journal_get_catalog",
            "sd_journal_get_catalog_for_message_id", "sd_journal_get_cursor", "sd_journal_get_cutoff_monotonic_usec",
            "sd_journal_get_cutoff_realtime_usec", "sd_journal_get_data", "sd_journal_get_data_threshold", "sd_journal_get_events",
            "sd_journal_get_fd", "sd_journal_get_monotonic_usec", "sd_journal_get_realtime_usec", "sd_journal_get_seqnum",
            "sd_journal_get_timeout", "sd_journal_get_usage", "sd_journal_has_persistent_files", "sd_journal_has_runtime_files",
            "sd_journal_next", "sd_journal_next_skip", "sd_journal_open", "sd_journal_open_container", "sd_journal_open_directory",
            "sd_journal_open_directory_fd", "sd_journal_open_files", "sd_journal_open_files_fd", "sd_journal_open_namespace",
            "sd_journal_perror", "sd_journal_perror_with_location", "sd_journal_previous", "sd_journal_previous_skip",
            "sd_journal_print", "sd_journal_printv", "sd_journal_printv_with_location", "sd_journal_print_with_location",
            "sd_journal_process", "sd_journal_query_unique", "sd_journal_reliable_fd", "sd_journal_restart_data",
            "sd_journal_restart_fields", "sd_journal_restart_unique", "sd_journal_seek_cursor", "sd_journal_seek_head",
            "sd_journal_seek_monotonic_usec", "sd_journal_seek_realtime_usec", "sd_journal_seek_tail", "sd_journal_send",
            "sd_journal_sendv", "sd_journal_sendv_with_location", "sd_journal_send_with_location", "sd_journal_set_data_threshold",
            "sd_journal_step_one", "sd_journal_stream_fd", "sd_journal_stream_fd_with_namespace", "sd_journal_test_cursor", "sd_journal_wait"
        ]
        
        implemented = ["sd_journal_print", "sd_journal_print_with_location", "sd_journal_send", "sd_journal_sendv", "sd_journal_sendv_with_location"]

        for js in journal_symbols:
            if js in implemented: continue
            if "print" in js or "send" in js or "perror" in js:
                f.write("int %s(const char* arg, ...) { sins_journal_log(arg); return 0; }\n" % js)
            elif "open" in js:
                f.write("int %s(void** j, ...) { return -ENOSYS; }\n" % js)
            else:
                f.write("int %s(void* j, ...) { return 0; }\n" % js)
        f.write("\n")

        # MANUAL BUS CALLER
        f.write("int sd_bus_call_method(void* bus, const char* dest, const char* path, const char* iface, const char* method, void* error, void** reply, const char* types, ...) {\n")
        f.write("    if (method && (strcmp(method, \"GetSession\") == 0 || strcmp(method, \"GetSeat\") == 0)) {\n")
        f.write("        void *m = NULL;\n")
        f.write("        int (*new_call)(void*, void**, const char*, const char*, const char*, const char*) = get_fn(\"sd_bus_message_new_method_call\");\n")
        f.write("        int (*append)(void*, char, const void*) = get_fn(\"sd_bus_message_append_basic\");\n")
        f.write("        int (*call)(void*, void*, uint64_t, void*, void**) = get_fn(\"sd_bus_call\");\n")
        f.write("        new_call(bus, &m, dest, path, iface, method);\n")
        f.write("        const char* arg = strcmp(method, \"GetSession\") == 0 ? \"self\" : \"seat0\";\n")
        f.write("        append(m, 's', &arg);\n")
        f.write("        return call(bus, m, 0, error, reply);\n")
        f.write("    }\n")
        f.write("    va_list aq; va_start(aq, types);\n")
        f.write("    int (*real_v)(void*, const char*, const char*, const char*, const char*, void*, void**, const char*, va_list) = get_fn(\"sd_bus_call_methodv\");\n")
        f.write("    int r = real_v(bus, dest, path, iface, method, error, reply, types, aq);\n")
        f.write("    va_end(aq); return r;\n}\n\n")

        # ULTRA SAFE ASM JUMPER MACRO
        f.write("#define JUMP_TO(name) \\\n")
        f.write("    static void* ptr_##name = NULL; \\\n")
        f.write("    __attribute__((naked)) void name() { \\\n")
        f.write("        __asm__( \\\n")
        f.write("            \"movq ptr_\" #name \"(%rip), %rax\\n\\t\" \\\n")
        f.write("            \"testq %rax, %rax\\n\\t\" \\\n")
        f.write("            \"jnz 1f\\n\\t\" \\\n")
        f.write("            \"pushq %rax\\n\\t\" \\\n")
        f.write("            \"pushq %rdi\\n\\t\" \\\n")
        f.write("            \"pushq %rsi\\n\\t\" \\\n")
        f.write("            \"pushq %rdx\\n\\t\" \\\n")
        f.write("            \"pushq %rcx\\n\\t\" \\\n")
        f.write("            \"pushq %r8\\n\\t\" \\\n")
        f.write("            \"pushq %r9\\n\\t\" \\\n")
        f.write("            \"pushq %r10\\n\\t\" \\\n")
        f.write("            \"pushq %r11\\n\\t\" \\\n")
        f.write("            \"subq $128, %rsp\\n\\t\" \\\n")
        f.write("            \"movaps %xmm0, 0(%rsp)\\n\\t\" \\\n")
        f.write("            \"movaps %xmm1, 16(%rsp)\\n\\t\" \\\n")
        f.write("            \"movq real_lib(%rip), %rdi\\n\\t\" \\\n")
        f.write("            \"leaq .Lfn_name_\" #name \"(%rip), %rsi\\n\\t\" \\\n")
        f.write("            \"call dlsym@PLT\\n\\t\" \\\n")
        f.write("            \"movq %rax, ptr_\" #name \"(%rip)\\n\\t\" \\\n")
        f.write("            \"movaps 16(%rsp), %xmm1\\n\\t\" \\\n")
        f.write("            \"movaps 0(%rsp), %xmm0\\n\\t\" \\\n")
        f.write("            \"addq $128, %rsp\\n\\t\" \\\n")
        f.write("            \"popq %r11\\n\\t\" \\\n")
        f.write("            \"popq %r10\\n\\t\" \\\n")
        f.write("            \"popq %r9\\n\\t\" \\\n")
        f.write("            \"popq %r8\\n\\t\" \\\n")
        f.write("            \"popq %rcx\\n\\t\" \\\n")
        f.write("            \"popq %rdx\\n\\t\" \\\n")
        f.write("            \"popq %rsi\\n\\t\" \\\n")
        f.write("            \"popq %rdi\\n\\t\" \\\n")
        f.write("            \"popq %rax\\n\\t\" \\\n")
        f.write("            \"movq ptr_\" #name \"(%rip), %rax\\n\\t\" \\\n")
        f.write("            \"1: jmp *%rax\\n\\t\" \\\n")
        f.write("            \".section .rodata\\n\\t\" \\\n")
        f.write("            \".Lfn_name_\" #name \": .string \\\"\" #name \"\\\"\\n\\t\" \\\n")
        f.write("            \".previous\" \\\n")
        f.write("        ); \\\n")
        f.write("    }\n\n")

        # MOCKS
        f.write("int sd_pid_get_session(pid_t pid, char **ret) { if (ret) *ret = strdup(\"self\"); return 0; }\n")
        f.write("int sd_session_is_active(const char *session) { return 1; }\n")
        f.write("int sd_session_get_seat(const char *session, char **ret) { if (ret) *ret = strdup(\"seat0\"); return 0; }\n")
        f.write("int sd_booted(void) { return 1; }\n\n")

        mock_names = ["sd_pid_get_session", "sd_session_is_active", "sd_session_get_seat", "sd_booted", "sd_bus_call_method"] + journal_symbols

        for s in all_names:
            if s in mock_names: continue
            f.write("JUMP_TO(%s)\n" % s)

    with open("gen_map.go", "w") as f:
        f.write("package main\nimport (\"fmt\"; \"os\")\nfunc main() {\n")
        f.write("\tf, _ := os.Create(\"pkg/libsystemd/libsystemd.map\")\n")
        f.write("\tdefer f.Close()\n")
        full_vers = sorted(vgroups.keys())
        last = ""
        for v in full_vers:
            vstr = "LIBSYSTEMD_%d" % v
            f.write("\tfmt.Fprintf(f, \"%s {\\n\")\n" % vstr)
            f.write("\tfmt.Fprintf(f, \"global:\\n\")\n")
            for s in sorted(set(vgroups[v])):
                f.write("\tfmt.Fprintf(f, \"        %s;\\n\")\n" % s)
            if not last:
                f.write("\tfmt.Fprintf(f, \"local:\\n        *;\\n};\\n\\n\")\n")
            else:
                f.write("\tfmt.Fprintf(f, \"} %s;\\n\\n\")\n" % last)
            last = vstr
        f.write("}\n")

if __name__ == '__main__':
    main()
