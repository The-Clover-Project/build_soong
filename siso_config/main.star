load("@builtin//encoding.star", "json")
load("@builtin//struct.star", "module")
load("./clang.star", "clang")
load("./java.star", "java")
load("./rust.star", "rust")

def init(ctx):
    # TODO: use handler for trivial step. e.g. stamp, copy

    step_config = {
        "platforms": {},
        "input_deps": {},
        "rules": [],
    }
    step_config = clang.step_config(ctx, step_config)
    step_config = java.step_config(ctx, step_config)
    step_config = rust.step_config(ctx, step_config)

    filegroups = {}
    filegroups.update(clang.filegroups(ctx))
    filegroups.update(java.filegroups(ctx))

    handlers = {}
    handlers.update(clang.handlers)
    handlers.update(java.handlers)
    handlers.update(rust.handlers)

    return module(
        "config",
        step_config = json.encode(step_config),
        filegroups = filegroups,
        handlers = handlers,
    )
