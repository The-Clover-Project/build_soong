load("@builtin//struct.star", "module")

def __config_get(ctx, key):
    if "config" in ctx.flags:
        for cfg in ctx.flags["config"].split(","):
            if cfg == key:
                return True
    return False

def __use_reclient(ctx):
    return __config_get(ctx, "reclient")

config = module(
    "config",
    use_reclient = __use_reclient,
)
