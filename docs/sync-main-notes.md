# Syncing with upstream `main`

The current repository snapshot does not include a configured remote. To pull the latest `main` branch:

1. Add the upstream remote, for example:
   ```bash
   git remote add origin https://github.com/a-voronkov/mattermost-bugsnag.git
   ```
2. Fetch the branch and merge it into the working branch:
   ```bash
   git fetch origin main
   git merge origin/main
   ```

If your environment blocks outbound HTTPS access, fetching will fail (e.g., `CONNECT tunnel failed, response 403`). In that case, repeat the fetch from a network that allows GitHub access or configure an internal mirror of the repository.
