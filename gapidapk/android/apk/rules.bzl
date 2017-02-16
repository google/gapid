load("//tools/build:rules.bzl", "extract", "filehash")

def gapid_apk(name= "", abi="", libs={}):
    natives = []
    for lib in libs:
        libname = name+"_"+lib
        natives += [":"+libname]
        extract(
            name = libname,
            zip = "{}:{}.apk".format(libs[lib],lib),
            entries = ["lib/{}/lib{}.so".format(abi, lib)],
            tags = ["manual"],
        )
    native.filegroup(
        name = name+"_resources",
        srcs = [
            "res/values/strings.xml",#TODO: substitute the strings.xml
        ],
        tags = ["manual"],
    )
    filehash(
        name = name+"_manifest",
        template = "AndroidManifest.xml.in",
        out = name + "/" + "AndroidManifest.xml",
        replace = "£{srchash}",
        srcs = natives + [
            "//gapidapk/android/app/src/main:gapid",
            ":"+name+"_resources",
        ],
        tags = ["manual"],
        visibility = ["//visibility:public"],
    )
    native.cc_library(
        name = name+"_native",
        linkstatic = 1,
        srcs = natives,
        tags = ["manual"],
    )
    native.android_binary(
        name = name,
        manifest_values = {
            "name": name,
            "abi": abi,
        },
        custom_package = "com.google.android.gapid",
        manifest = ":"+name+"_manifest",
        manifest_merger = "android",
        deps = [
            "//gapidapk/android/app/src/main:gapid",
            ":"+name+"_native",
        ],
        resource_files = [
            ":"+name+"_resources",
        ],
        visibility = ["//visibility:public"],
        tags = ["manual"],
    )
