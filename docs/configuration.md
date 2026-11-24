# Configuration Guide

If you do not plan to use the generated install manifest because you need some specific configurations, please read this documentation. The chart can be customized through the [values.yaml file](../chart/values.yaml) which contains default values. Each field is commented for you to know what to do with it. Once customized, you can render the chart using helm:

```
helm template chart/
```

When `controller.safeMode.permittedUserGroups` is nonempty, `controller.userInfoHook` must be `true`. The controller refuses to start otherwise, because group validation relies on the user info webhook to replace caller-supplied identity annotations with the authenticated request identity. This requirement also applies when `controller.safeMode.enable` is `false`, and is checked after command-line overrides are applied.
