load("@builtin//struct.star", "module")
load("./config.star", "config")

def __filegroups(ctx):
    if config.use_reclient(ctx):
        return {}
    fg = {
        "prebuilts/clang/host/linux-x86/clang-r563880/bin:bin": {
            "type": "glob",
            "includes": [
                "clang*",  # clang, clang++, clang-<ver> clang-real, clang++-real, clang++.real etc.
            ],
        },
        "prebuilts/clang/host/linux-x86/clang-r563880/include:include": {
            "type": "glob",
            "includes": [
                "*",
            ],
        },
        "prebuilts/clang/host/linux-x86/clang-r563880/lib:headers": {
            "type": "glob",
            "includes": [
                "*.h",
                "*_ignorelist.txt",
            ],
        },
        "prebuilts/clang/host/linux-x86/clang-r563880/android_libc++/ndk:headers": {
            "type": "glob",
            "includes": [
                "*.h",
                "*/include/c++/v1/*",
            ],
        },
        "prebuilts/gcc/linux-x86/host/x86_64-linux-glibc2.17-4.8/sysroot/usr/include:include": {
            "type": "glob",
            "includes": [
                "*",
            ],
        },
        "prebuilts/gcc/linux-x86/host/x86_64-linux-glibc2.17-4.8/sysroot/usr/lib:headers": {
            "type": "glob",
            "includes": [
                "*.h",
                "crtbegin.o",
            ],
        },
        "bionic/libc/include:headers": {
            "type": "glob",
            "includes": [
                "*.h",
            ],
        },
        "bionic/libc/kernel/uapi:headers": {
            "type": "glob",
            "includes": [
                "*.h",
            ],
        },
        "bionic/libc/kernel/android/uapi:headers": {
            "type": "glob",
            "includes": [
                "*.h",
            ],
        },
        "prebuilts/gcc/linux-x86/host/x86_64-w64-mingw32-4.8/x86_64-w64-mingw32/include:include": {
            "type": "glob",
            "includes": ["*"],
        },
        "prebuilts/gcc/linux-x86/host/x86_64-w64-mingw32-4.8/x86_64-w64-mingw32/lib:lib": {
            "type": "glob",
            "includes": ["crtbegin.o"],
        },
        "prebuilts/gcc/linux-x86/host/x86_64-w64-mingw32-4.8/x86_64-w64-mingw32/lib32:lib32": {
            "type": "glob",
            "includes": ["crtbegin.o"],
        },
        "external/boringssl/src/include:include": {
            "type": "glob",
            "includes": ["*.h"],
        },
        # prebuilts/gcc/linux-x86/host/x86_64-linux-glibc2.17-4.8/sysroot/usr/include includes e.g. linux/rtnetlink.h
        "external/libbpf/include/uapi:headers": {
            "type": "glob",
            "includes": ["*.h"],
        },
    }
    return fg

__handlers = {}

def __step_config(ctx, step_config):
    if config.use_reclient(ctx):
        step_config["rules"].extend([
            {
                "name": "g.cc.cc",
                "action": "g.cc.cc",
                "timeout": "2m",
                "use_remote_exec_wrapper": True,
                # "debug": True,
            },
        ])
        return step_config

    step_config["rules"].extend([
        {
            "name": "g.cc.cc",
            "action": "g.cc.cc",
            "remote": True,
            "canonicalize_dir": True,
            "timeout": "2m",
            # "debug": True,
        },
    ])
    step_config["input_deps"].update({
        "prebuilts/clang/host/linux-x86/clang-r563880:headers": [
            "prebuilts/clang/host/linux-x86/clang-r563880/android_libc++/ndk:headers",
            "prebuilts/clang/host/linux-x86/clang-r563880/bin:bin",
            "prebuilts/clang/host/linux-x86/clang-r563880/include:include",
            "prebuilts/clang/host/linux-x86/clang-r563880/lib:headers",
        ],
        "prebuilts/gcc/linux-x86/host/x86_64-linux-glibc2.17-4.8/sysroot:headers": [
            "prebuilts/gcc/linux-x86/host/x86_64-linux-glibc2.17-4.8/sysroot/usr/include:include",
            "prebuilts/gcc/linux-x86/host/x86_64-linux-glibc2.17-4.8/sysroot/usr/lib:headers",
        ],
        "prebuilts/gcc/linux-x86/host/x86_64-w64-mingw32-4.8/x86_64-w64-mingw32:headers": [
            "prebuilts/gcc/linux-x86/host/x86_64-w64-mingw32-4.8/x86_64-w64-mingw32/include:include",
            "prebuilts/gcc/linux-x86/host/x86_64-w64-mingw32-4.8/x86_64-w64-mingw32/lib:lib",
            "prebuilts/gcc/linux-x86/host/x86_64-w64-mingw32-4.8/x86_64-w64-mingw32/lib32:lib32",
            "prebuilts/gcc/linux-x86/host/x86_64-w64-mingw32-4.8/x86_64-w64-mingw32/lib64",
        ],
        # TODO(b/433611110): --sysroot out/soong/ndk/sysroot
        "out/soong/ndk/sysroot:headers": [
            "out/soong/ndk_headers.timestamp:inputs",
        ],

        # asm include
        "device/google/cuttlefish/host/libs/confui/fonts.S": [
            # .incbin
            "device/google/cuttlefish/host/libs/confui/Roboto-Medium.ttf",
            "device/google/cuttlefish/host/libs/confui/Roboto-Regular.ttf",
            "device/google/cuttlefish/host/libs/confui/Shield.ttf",
        ],
    })
    step_config["inputs_requiring_clang_scandeps"].extend([
        # #include LIB_INCLUDE(SYM_LIB, Sym)
        "external/ms-tpm-20-ref/TPMCmd/tpm/include/Tpm.h",

        # #include RELOC_TYPESE
        "external/elfutils/backends/common-reloc.c",

        # #if __has_include(<android/binder_ibinder.h>)
        "frameworks/native/libs/binder/ndk/include_cpp/android/binder_to_string.h",

        # #include PATH(android/hardware/audio/common/COMMON_TYPES_FILE_VERSION/types.h)
        "hardware/interfaces/audio/common/all-versions/default/HidlUtils.h",

        # #include PATH(android/hardware/audio/common/COMMON_TYPES_FILE_VERSION/types.h)
        "hardware/interfaces/audio/common/all-versions/default/UuidUtils.h",

        # #include PATH(APM_XSD_ENUMS_H_FILENAME)
        "hardware/interfaces/audio/common/all-versions/default/7.0/HidlUtils.cpp",

        # #include PREFIX(android/media/audio/common/AudioAttributes.h)
        "frameworks/av/media/audioaidlconversion/include/media/AidlConversionCppNdk-impl.h",

        # #include PATH(android/hardware/audio/CORE_TYPES_FILE_VERSION/types.h)
        "frameworks/av/media/libaudiohal/impl/CoreConversionHelperHidl.h",

        # #include PATH(android/hardware/audio/effect/COMMON_TYPES_FILE_VERSION/types.h)
        "frameworks/av/media/libaudiohal/impl/EffectConversionHelperHidl.h",

        # #include PATH(android/hardware/audio/effect/FILE_VERSION/types.h)
        "frameworks/av/media/libaudiohal/impl/EffectBufferHalHidl.h",
    ])
    return step_config

clang = module(
    "clang",
    step_config = __step_config,
    filegroups = __filegroups,
    handlers = __handlers,
)
