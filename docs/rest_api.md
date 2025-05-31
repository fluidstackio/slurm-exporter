# Querying the Slurm Rest API

```bash
sudo scontrol token
```

```bash
export SLURM_JWT=xyz
```

```bash
curl -H "X-SLURM-USER-TOKEN:$SLURM_JWT" -H "X-SLURM-USER-NAME:$(whoami)" http://localhost:6820/slurm/v0.0.41/jobs
```
