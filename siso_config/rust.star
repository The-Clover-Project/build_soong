load("@builtin//struct.star", "module")
load("./config.star", "config")

def __filegroups(ctx):
    return {}

__handlers = {}

def __step_config(ctx, step_config):
    step_config["rules"].extend([
        {
            "name": "g.rust.rustc",
            "action": "g.rust.rustcRE",
            "timeout": "2m",
            "use_remote_exec_wrapper": True,
        },
    ])
    return step_config

rust = module(
    "rust",
    step_config = __step_config,
    filegroups = __filegroups,
    handlers = __handlers,
)
