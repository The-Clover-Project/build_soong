load("@builtin//struct.star", "module")
load("./config.star", "config")

def __filegroups(ctx):
    return {}

__handlers = {}

def __step_config(ctx, step_config):
    if config.use_reclient(ctx):
        step_config["rules"].extend([
            {
                "name": "g.java.d8",
                "action": "g.java.d8RE",
                "timeout": "2m",
                "use_remote_exec_wrapper": True,
            },
            {
                "name": "g.java.javac",
                "action": "g.java.javacRE",
                "timeout": "2m",
                "use_remote_exec_wrapper": True,
            },
            {
                "name": "g.java.r8",
                "action": "g.java.r8RE",
                "timeout": "2m",
                "use_remote_exec_wrapper": True,
            },
        ])
        return step_config

    # TODO: siso native remote
    return step_config

java = module(
    "java",
    step_config = __step_config,
    filegroups = __filegroups,
    handlers = __handlers,
)
