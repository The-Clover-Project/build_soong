load("@builtin//struct.star", "module")
load("./config.star", "config")

def __filegroups(ctx):
    return {}

__handlers = {}

def __step_config(ctx, step_config):
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

clang = module(
    "clang",
    step_config = __step_config,
    filegroups = __filegroups,
    handlers = __handlers,
)
