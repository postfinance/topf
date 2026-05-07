customization:
    extraKernelArgs:
        - some-node-specific-arg-{{.Node.Host}}
    systemExtensions:
        officialExtensions:
            - siderolabs/vmtoolsd-guest-agent
