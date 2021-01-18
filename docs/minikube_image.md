# Custom Minikube Image

## Description

Some of the available disruptions (like `network_latency` and `network_limitation`) rely on the [`tc` command](http://man7.org/linux/man-pages/man8/tc.8.html) to function, which sets `qdiscs` on network devices in order to enforce extra network latency or bandwidth limitations.

These functions of `tc` rely on certain linux kernel modules being available, namely `SCH_TBF`, `SCH_NETEM`, and `SCH_PRIO`, the last of which isn't included in the base Minikube image that you'd get by installing and running Minikube normally. Thus, when testing changes locally, these disruptions wouldn't be able to operate and would crash with an opaque error under a standard Minikube cluster.

To work around this, we've created a custom Minikube ISO that contains not only `SCH_TBF` and `SCH_NETEM` (like the standard image) but also `SCH_PRIO` (which allows the latency disruption to work) and uploaded this image to S3. Then, passing in this image as the `--iso-url` parameter in `minikube start` solves the problem.

## Creating the Image

```
cd hack
sed -i'' -e "s/my-ip/$(curl ifconfig.me)/" packer.json
```

Fill out values in `packer.json` for _profile_, _vpc_id_ and _subnet_id_

```
aws-vault exec staging-engineering -- packer build packer.json
```

The process of building the minikube ISO can take a couple hours. Be patient.

The completed minikube ISO will be copied to your local host at /tmp/minikube.iso

## Uploading the Image

> Note: Only Datadog employees currently have permissions to upload to the `public-chaos-controller` S3 bucket.

Finally, `aws s3 cp` the new ISO image into the `s3://public-chaos-controller/minikube/` directory, preferably using the `minikube-YYYY-MM-DD.iso` versioning format so as to avoid affecting existing, known-working images.

Then, update the `Makefile` of this repository to use the new image, once you've verified it works.
