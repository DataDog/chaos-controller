# gRPC Disruption Inceptor

When the interceptor recognizes a query's endpoint as one which is actively getting disrupted, the interceptor generates a random integer from `0` to `100`, and consults a hasmap for the `PercentToAlteration` mapping. The hashmap is represented as an array of 100 or less items, and provided the random integer, it identifies the disruption to apply to a query. The user cannot sepcify queryPercents for a single endpoint which sum to over `100%`. You can see an example below of a hashmap that does not define all 100% of possible requests. You can see the next section for an example of a mapping which only has 25 entires.

## gRPC Disruption - Algorithm Examples

### Multiple alterations with defined queryPercent

The following is a complex example to illustrate the algorithm's behavior.

```
spec:
  grpc:
    endpoints:
      - endpoint: /chaos_dogfood.ChaosDogfood/order
        override: "{}"
        queryPercent: 5
      - endpoint: /chaos_dogfood.ChaosDogfood/order
        error: NOT_FOUND
        queryPercent: 5
      - endpoint: /chaos_dogfood.ChaosDogfood/order
        error: PERMISSION_DENIED
        queryPercent: 15
```

For the above specs, the calculated `PercentToAlteration` would look something like:

```
PercentToAlteration {
    0  -> Override: {}
    1  -> Override: {}
    2  -> Override: {}
    3  -> Override: {}
    4  -> Override: {}
    5  -> Error: NOT_FOUND
    6  -> Error: NOT_FOUND
    7  -> Error: NOT_FOUND
    8  -> Error: NOT_FOUND
    9  -> Error: NOT_FOUND
    10 -> Error: PERMISSION_DENIED
    11 -> Error: PERMISSION_DENIED
    12 -> Error: PERMISSION_DENIED
    13 -> Error: PERMISSION_DENIED
    .. -> Error: ...
    22 -> Error: PERMISSION_DENIED
    23 -> Error: PERMISSION_DENIED
    24 -> Error: PERMISSION_DENIED
}
```

In this case, we would return an Override with empty results for 5% of queries, a `NOT_FOUND` error for 5% of queries, and return `PERMISSION_DENIED` for 15% of queries.

### Multiple alterations with some undefined queryPercents

We may also be provided with a configuration where some set of `queryPercent`s are defined, but not all..

```
spec:
  grpc:
    endpoints:
      - endpoint: /chaos_dogfood.ChaosDogfood/order
        override: "{}"
        queryPercent: 25
      - endpoint: /chaos_dogfood.ChaosDogfood/order
        error: NOT_FOUND
      - endpoint: /chaos_dogfood.ChaosDogfood/order
        error: PERMISSION_DENIED
```

As in the previous case, all alterations with a defined `queryPercent` are allocated upfront. The algorithm keeps track of alterations which do not yet have `queryPercent`s assigned, and splits the remaining (unconfigured) queries equally amongst these unassigned alterations.

```
PercentToAlteration {
    0   -> Override: {}
    1   -> Override: {}
    2   -> Override: {}
    3   -> Override: {}
    4   -> Override: {}
    5   -> Override: {}
    6   -> Override: {}
    ..  -> Override: ..
    22  -> Override: {}
    23  -> Override: {}
    24  -> Override: {}
    25  -> Error: NOT_FOUND
    26  -> Error: NOT_FOUND
    26  -> Error: NOT_FOUND
    ..  -> Error: ...
    61  -> Error: NOT_FOUND
    62  -> Error: NOT_FOUND
    63  -> Error: PERMISSION_DENIED
    64  -> Error: PERMISSION_DENIED
    65  -> Error: PERMISSION_DENIED
    ..  -> Error: ...
    99  -> Error: PERMISSION_DENIED
    100 -> Error: PERMISSION_DENIED
}
```

### Simpler case of undefined queryPercents

You may have noted that the second example appears a tad complex. The intuition behind this design is to support the case where a user wants to quickly define a disruption which errors on all queries (replicating a bad roll out). For one error, the algorithm returns an error every time. For two errors, the algorithm returns half of the queries with the first error and half with the other.

```
spec:
  grpc:
    endpoints:
      - endpoint: /chaos_dogfood.ChaosDogfood/order
        error: NOT_FOUND
      - endpoint: /chaos_dogfood.ChaosDogfood/order
        error: PERMISSION_DENIED
```

Rather than constraining the user in how they mix and match this simple configuration style with the explicit `spec.gprc.endpoints[x].queryPercent` field, the current implementation would simply do its best to apply of the configurations.

```
PercentToAlteration {
    0   -> Override: {}
    1   -> Override: {}
    2   -> Override: {}
    3   -> Override: {}
    4   -> Override: {}
    5   -> Override: {}
    6   -> Override: {}
    ..  -> Override: ..
    22  -> Override: {}
    23  -> Override: {}
    24  -> Override: {}
    25  -> Error: NOT_FOUND
    26  -> Error: NOT_FOUND
    26  -> Error: NOT_FOUND
    ..  -> Error: ...
    61  -> Error: NOT_FOUND
    62  -> Error: NOT_FOUND
    63  -> Error: PERMISSION_DENIED
    64  -> Error: PERMISSION_DENIED
    65  -> Error: PERMISSION_DENIED
    ..  -> Error: ...
    99  -> Error: PERMISSION_DENIED
    100 -> Error: PERMISSION_DENIED
}
```

## Design Implications

### Setting 0 as queryPercent

Users are prevented from setting `queryPercent: 0`. Because the Kubebuilder infers 0 for omitted `int`s, this configuration is synonymous for the chaos-controller to not setting a `querypercent` at all. This means that the alteration with this `queryPercent` setting would actually be applied to the remaining queries that have not yet be claimed by another alteration. There isn't a straight forward way to chaos-controller to catch this for users so we validate inputs to be between 1 and 100.

### Many errors, but very few slots remaining

When an even split across remaining points is not possible. For example, if 9% of queries are unaccounted for, and there are 6 different errors to assign to the hashmap, the `pctPerAlt` (describing the hashmap each query should be assign) would be `9 / 6` which is `1` in integer division. The final mapping would look like:
```
{
	..  -> ...
	90  -> ERROR_X
	91  -> ERROR_X
	92  -> ERROR_1
	93  -> ERROR_2
	94  -> ERROR_3
	95  -> ERROR_4
	96  -> ERROR_5
	97  -> ERROR_6
	98  -> ERROR_6
	99  -> ERROR_6
	100 -> ERROR_6
}
```
Note that the final alteration (in this case `ERROR_6`, always covers the remaining `Percent`s up to and including 100. This can result in a very weird proportions where there are not a lot of `Percent`s left. Although these outcomes are unintuitive and therefore not very user-friendly, the common usecase for these disruptions is so 