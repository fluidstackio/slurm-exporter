```fish
sudo scontrol token
```

```fish
set -x SLURM_JWT <token>
```

```fish
curl -H X-SLURM-USER-TOKEN:$SLURM_JWT -X GET 'http://127.0.0.1:6820/slurm/v0.0.41/nodes'
```

The list of API endpoints is [here](https://slurm.schedmd.com/rest_api.html).

