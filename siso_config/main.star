load("@builtin//encoding.star", "json")
load("@builtin//path.star", "path")
load("@builtin//struct.star", "module")
load("./clang.star", "clang")
load("./java.star", "java")
load("./rust.star", "rust")

__my_imports = [clang, java, rust]

def __filegroups(ctx, vars, filegroups):
    for i in __my_imports:
        filegroups.update(i.filegroups(ctx, vars))
    return filegroups

def __handlers(ctx, vars, handlers):
    for i in __my_imports:
        handlers.update(i.handlers)
    return handlers

def __step_config(ctx, vars, step_config):
    if vars.use_reclient:
        step_config["rules"].extend([
            {
                "name": "rewrapper",
                "command_prefix": "prebuilts/remoteexecution-client/live//rewrapper ",
                "timeout": "2m",
                "use_remote_exec_wrapper": True,
            },
        ])
    for i in __my_imports:
        step_config = i.step_config(ctx, vars, step_config)
    return step_config

def __generate(ctx, vars, dir_modules):
    step_config = {
        "platforms": {},
        "input_deps": {},
        "inputs_requiring_clang_scandeps": [],
        "rules": [],
    }
    filegroups = {}
    handlers = {}

    if vars.use_rbe:
        step_config["platforms"] = {
            "default": {
                "OSFamily": "Linux",
                "container-image": vars.RBE_container_image,
                "Pool": "default",
            },
            "java16": {
                "OSFamily": "Linux",
                "container-image": vars.RBE_container_image,
                "Pool": "java16",
            },
        }

    for m in dir_modules:
        step_config = m.step_config(ctx, vars, step_config)
        filegroups = m.filegroups(ctx, vars, filegroups)
        handlers = m.handlers(ctx, vars, handlers)

    # for action sandboxed build
    action_sandbox = False
    if "config" in ctx.flags:
        action_sandbox = "action_sandbox" in ctx.flags["config"].split(",")
    if action_sandbox and vars.nsjail_path:
        step_config["sandbox"] = {
            "backend": "nsjail",
            "nsjail_path": vars.nsjail_path,
            "nsjail_workdir": path.join(vars.OUT_DIR, "soong", ".siso_action_sandboxing_workdir"),
            "nsjail_outdir": vars.OUT_DIR,
        }
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
