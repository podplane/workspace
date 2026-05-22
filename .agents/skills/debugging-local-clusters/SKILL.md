---
name: debugging-local-clusters
description: "Debugs Podplane local development VMs and local clusters. Use when asked to debug a Podplane local VM, local cluster, local dev environment, cloud-init, user-data, vmconfig, or systemd services inside a VM."
---

# Debugging Podplane Local Clusters

Use this skill to debug a Podplane local development VM or local cluster from the `podplane/podplane` CLI repository.

## Starting Point

Prefer the adjacent workspace layout:

```text
workspace/
├── podplane/
│   ├── podplane/    # run CLI commands here
│   ├── vmconfig/
│   └── components/
```

Run local VM commands from the Podplane CLI checkout:

```sh
go run . local status
go run . local shell
```

If a named local VM is involved, pass the name/id consistently to every `local` command.

## Debugging Workflow

1. Check the VM from the host:

   ```sh
   go run . local status
   ```

   If SSH is unavailable or early boot is suspected, use the serial console:

   ```sh
   go run . local console
   ```

2. Check cloud-init and user-data first:

   ```sh
   go run . local shell "cloud-init status --long"
   go run . local shell "sudo journalctl -u cloud-init -u cloud-final --no-pager -n 200"
   go run . local shell "sudo test -f /var/log/cloud-init-output.log && sudo tail -n 200 /var/log/cloud-init-output.log"
   ```

   Confirm whether `user-data.sh` completed, whether dependencies downloaded, and whether vmconfig was intentionally skipped for development manifests.

   Also check whether local vmconfig has actually been synced/installed into the VM. Development manifests can make cloud-init exit successfully before Kubernetes is installed, waiting for the dev to run `make knc-sync` or `make knc-watch` from `../vmconfig`:

   ```sh
   go run . local shell "sudo test -x /opt/podplane/bin/install.sh && echo 'vmconfig scripts synced' || echo 'vmconfig scripts not synced'"
   go run . local shell "sudo test -f /opt/podplane/share/vmconfig-installed.json && echo 'vmconfig installed' || echo 'vmconfig not installed'"
   go run . local shell "sudo ls -la /opt/podplane/bin /opt/podplane/share 2>/dev/null || true"
   ```

   If `/opt/podplane/bin/install.sh` is missing or `/opt/podplane/share/vmconfig-installed.json` is absent after a dev manifest boot, run one of:

   ```sh
   cd ../vmconfig
   make knc-sync
   make knc-watch
   ```

   `knc-sync` syncs local vmconfig once; `knc-watch` is preferred by devs while iterating because it syncs and applies install/configure/restart on changes, but if you run `make knc-watch` you probably want to do it on a short timeout unless you explicitly want the live-reload functionality.

3. Inspect Podplane/vmconfig state files:

   ```sh
   go run . local shell "sudo ls -la /opt/podplane /opt/podplane/etc /opt/podplane/share 2>/dev/null || true"
   go run . local shell "sudo test -f /opt/podplane/etc/user-data.env && sudo sed -n '1,160p' /opt/podplane/etc/user-data.env"
   go run . local shell "sudo test -f /opt/podplane/etc/detected.env && sudo sed -n '1,160p' /opt/podplane/etc/detected.env"
   go run . local shell "sudo test -f /opt/podplane/etc/mutable.env && sudo sed -n '1,160p' /opt/podplane/etc/mutable.env"
   ```

4. Check systemd from the bottom of the stack upward. Start with `nstance-agent`, then inspect the rest:

   ```sh
   go run . local shell "systemctl --failed --no-pager"
   go run . local shell "systemctl status nstance-agent --no-pager"
   go run . local shell "sudo journalctl -u nstance-agent --no-pager -n 200"
   go run . local shell "systemctl list-units --type=service --state=running,failed --no-pager"
   ```

   Then check relevant services such as `containerd`, `kubelet`, `netsy`, `kube-apiserver`, `kube-scheduler`, `kube-controller-manager`, `registry`, and other units present on the VM:

   ```sh
   go run . local shell "systemctl status <service> --no-pager"
   go run . local shell "sudo journalctl -u <service> --no-pager -n 200"
   ```

5. Check Kubernetes only after the VM services look healthy:

   ```sh
   go run . local shell "sudo crictl ps -a || true"
   go run . local shell "sudo crictl logs <container-id>"
   go run . local shell "sudo journalctl -u kubelet --no-pager -n 200"
   ```

6. Verify host-side local cluster access and ingress routing:

   ```sh
   kubectl config get-contexts | grep -E '(^|[[:space:]])local([[:space:]]|$)' || true
   kubectl config view --raw --minify --context local
   kubectl --context local get --raw /readyz?verbose
   kubectl --context local get nodes -o wide
   kubectl --context local get pods -A -o wide
   ```

   Confirm that the local context points at the local ingress Kubernetes API hostname, not the VM's raw forwarded port. For the default cluster this should be `https://default.k8s.localhost:4433`, with a `podplane hooks kubectl-auth --cluster default --user test-user` exec credential.

   If host-side kubectl reports an exec credential error such as `no cluster config found`, inspect and re-login with the generated local cluster config:

   ```sh
   sed -n '1,160p' ~/.podplane/data/local/default/cluster.jsonc
   go run . hooks kubectl-auth --cluster default --user test-user
   go run . login --headless -f ~/.podplane/data/local/default/cluster.jsonc
   ```

   Then curl both paths through the host local ingress proxy. The Kubernetes endpoint should proxy to the VM API server, and the Traefik endpoint should either reach Traefik or clearly fail because Traefik/components are not installed yet:

   ```sh
   curl -vk --resolve default.k8s.localhost:4433:127.0.0.1 https://default.k8s.localhost:4433/readyz
   curl -vk --resolve default.localhost:4433:127.0.0.1 https://default.localhost:4433/
   ```

   For a non-default local cluster, replace `default` in the hostnames and paths with the cluster ID used for `go run . local --id <id> ...`.

## Development Guide Notes

When the local VM was created after:

```sh
go run . deps download \
  --components ../components/manifests/components.json \
  --vmconfig ../vmconfig/manifests/knc.debian-13.arm64.json
```

the vmconfig manifest may intentionally contain an unreleased vmconfig stub. In that mode, user-data skips installing a prebuilt vmconfig tarball so that `make knc-watch` from `../vmconfig` can sync local vmconfig into the VM and run install/configure/restart.

Use this to continue debugging vmconfig changes:

```sh
cd ../vmconfig
make knc-watch
```

If you need to test first-install behavior again, stop and remove the VM, then recreate it:

```sh
go run . local stop --rm
go run . local start
```

## Reporting Findings

When reporting back, summarize:

- whether the VM booted and whether cloud-init/user-data succeeded
- the first failing service in dependency order, starting from `nstance-agent`
- the key journal error lines, not the entire log
- whether the issue appears to be CLI local VM orchestration, dependency cache/manifest content, vmconfig install/configure, or Kubernetes/component bootstrap
