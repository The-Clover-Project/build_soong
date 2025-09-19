load("@builtin//encoding.star", "json")
load("@builtin//struct.star", "module")
load("./clang.star", "clang")
load("./java.star", "java")
load("./rust.star", "rust")

__my_imports = [clang, java, rust]

def __filegroups(ctx, filegroups):
    for i in __my_imports:
        filegroups.update(i.filegroups(ctx))
    return filegroups

def __handlers(ctx, handlers):
    for i in __my_imports:
        handlers.update(i.handlers)
    return handlers

def __step_config(ctx, step_config):
    for i in __my_imports:
        step_config = i.step_config(ctx, step_config)
    return step_config

def __generate(ctx, dir_modules):
    step_config = {
        "platforms": {},
        "input_deps": {},
        "inputs_requiring_clang_scandeps": [],
        "rules": [],
    }
    filegroups = {}
    handlers = {}

    for m in dir_modules:
        step_config = m.step_config(ctx, step_config)
        filegroups = m.filegroups(ctx, filegroups)
        handlers = m.handlers(ctx, handlers)

    return module(
        "config",
        step_config = json.encode(step_config),
        filegroups = filegroups,
        handlers = handlers,
    )

main = module(
    "main",
    step_config = __step_config,
    filegroups = __filegroups,
    handlers = __handlers,
    generate = __generate,
)

def init(ctx):
    # TODO: use handler for trivial step. e.g. stamp, copy

    dir_modules = [main]
    return __generate(ctx, dir_modules)
