# CTF Pilot's CTFd Manager

CTF Pilot's CTFd Manager is a management API and orhestration tool for [CTFd](https://ctfd.io).

The tool contains a management API that allows for programmatic management of CTFd content, and a Kubernetes listener, that allows for continued deployment of CTFd content.

> [!NOTE]
> Currently, initial setup of CTFd, and continued deployment of challenges and pages is supported.

## How to run

The following sections describes how to run the CTFd manager application.

### Notes

The application makes use of restarting to try to fix errors.  
Ensure to run the application with health checks and restart when appropriate automatically.

### Prerequisites

The application is built to run *inside* a Kubernetes cluster.  
This is due to how it retrieves the Kubernetes connection configuration.

#### Service account

The application requires the following access through Kubernetes RBAC service account:

- Api groups: `""`, resources: `configmaps`, verbs: `get`, `list`, `watch`, `update`, `patch`

#### ConfigMaps

In order for the application to run properly, the following ConfigMaps must be created in the same namespace as the application is running in, before the application is started:

- `ctfd-access-token`: Will contain the CTFd API access token for the manager. The manager will automatically store the API access token in `access_token` when setting up CTFd for the first time.
- `ctfd-challenges`: Will store the uploaded challenges to CTFd. The manager will automatically create and update this ConfigMap when challenges are uploaded the service.
- `ctfd-pages`: Will store the uploaded pages to CTFd. The manager will automatically create and update this ConfigMap when pages are uploaded through the service.
- `challenge-configmap-hashset`: Will store a hashset of uploaded challenges and pages, in order to track changes. The manager will automatically create and update this ConfigMap when challenges or pages are uploaded through the service.
- `mapping-map`: Should store a mapping of category and difficulty slugs to category names. Will be used to dynamically change the "category" field in challenges. See the [Category and Difficulty Mapping](#category-and-difficulty-mapping) section for more information.

### Running the application

> [!TIP]
> Included in the repository is a sample Kubernetes deployment manifest, `k8s/example.yaml`, which can be used to deploy the application to a Kubernetes cluster.

> [!CAUTION]
> Currently, the application has not been built to run with multiple replicas.  
> Ensure to only run a single replica of the application to avoid race conditions and conflicts.

The repository automatically builds a Docker image for the application and pushes it to GitHub Container Registry: `ghcr.io/ctfpilot/ctfd-manager:latest`.  
The image is versioned, so you can also use specific versions, for example: `ghcr.io/ctfpilot/ctfd-manager:1.0.0`.  
For more information, see the [Docker image package](https://github.com/ctfpilot/ctfd-manager/pkgs/container/ctfd-manager)

#### Configuration

> [!WARNING]
> Before configuring, please ensure permissions is created correctly through the service account, and the required ConfigMaps are created. See the [Prerequisites](#prerequisites) section for more information.  
> Failure to setup these elements correctly may lead to the application failing to start or work correctly.

The application needs the following configuration through environment variables:

- `CTFD_URL`: The URL to the CTFd instance to manage.  
  *Example: `https://ctfd.example.com`*
- `PASSWORD`: The password to use for API authorization, provided by the service.  
  **This should be a strong password, as it will give admin access to the CTFd instance and all challenge information.**
  *Example: `SuperSecretPassword123!`*
- `NAMESPACE`: The Kubernetes namespace the application is running in.  
  *Example: `ctfd`*
- `GITHUB_REPO`: The GitHub repository to use for challenge files.  
  *Example: `ctfpilot/ctfd-challenges`*
- `GITHUB_BRANCH`: The GitHub branch to use for challenge files.  
  *Example: `main` or `develop`*
- `GITHUB_TOKEN`: The GitHub access token to use for accessing the challenge files repository.  
  *Example: `ghp_XXXXXXXXXXXXXXXXXXXX`*

> [!IMPORTANT]
> Passwords should always be stored using secrets instead of cleartext environment variables.

#### Development

To develop, build the docker image locally:

```bash
docker build -t ctfd-manager:dev .
```

Then deploy the application within Kubernetes, using the local image.

To update the Kubernetes deployment file, update the deployment template in `template/k8s.yml`.  
An updated `k8s/k8s.yml` will then be automatically generated on the next release.

## Operation guide

The following sections describes how to operate the CTFd manager.

### Category and Difficulty Mapping

In order to get proper categories and difficulties in CTFd, categories and difficulties can be mapped to specific names through the `mapping-map` ConfigMap.

Three mappings are available:

- `categories`: A mapping of category slugs to category names.  
  *Example: `web: Web Challenges`*
- `difficulties`: A mapping of difficulty slugs to difficulty names.  
  *Example: `easy: Easy Challenges`*
- `difficulty-categories`: A mapping of difficulty slugs to category names.  
  Allows for mapping a difficulty to a specific category, such as `beginner` difficulty challenges to a "Beginner Challenges" category.
  *Example: `easy: Easy Challenges`*

For category names, the service will first check the `difficulty-categories` mapping, then the `categories` mapping.  
If no mapping is found, the original category name from the challenge file will be used.  
If no category is found (empty string for challenge category), the challenge will be placed in the "Uncategorized" category in CTFd.

Currently, the difficulty is not uploaded to CTFd, as CTFd does not have a built-in difficulty field for challenges.

### Attaching the manager to an existing CTFd

In order to attach the manager to an existing CTFd instance, you need to provide the manager with a valid CTFd API access token.  
This can be done by adding the access token to the `ctfd-access-token` ConfigMap in the same namespace as the manager is running in.

The token must be added under the key `access_token`.

For existing challenges and pages, these can be added to their appropriate ConfigMaps (`ctfd-challenges` and `ctfd-pages`) in order for the manager to manage them.  
They are added in the format of: `<challenge-or-page-slug>: <ctfd-id>`.

### Health checks

In order to ensure the application is running correctly, it exposes two health check endpoints, which provides the same information: `status` and  `/api/status`.  
They will return 200 OK when the application is running correctly, and 500 Internal Server Error when there is an issue.  
The return body will contain a JSON object with a `status` field, which will be either `ok` or `error`, and in case of an error.

Please set up Kubernetes to listen on these endpoints for liveness and readiness probes.

## Contributing

We welcome contributions of all kinds, from **code** and **documentation** to **bug reports** and **feedback**!

Please check the [Contribution Guidelines (`CONTRIBUTING.md`)](/CONTRIBUTING.md) for detailed guidelines on how to contribute.

To maintain the ability to distribute contributions across all our licensing models, **all code contributions require signing a Contributor License Agreement (CLA)**.
You can review **[the CLA here](https://github.com/ctfpilot/cla)**. CLA signing happens automatically when you create your first pull request.  
To administrate the CLA signing process, we are using **[CLA assistant lite](https://github.com/marketplace/actions/cla-assistant-lite)**.

*A copy of the CLA document is also included in this repository as [`CLA.md`](CLA.md).*  
*Signatures are stored in the [`cla` repository](https://github.com/ctfpilot/cla).*

## License

This schema and repository is licensed under the **EUPL-1.2 License**.  
You can find the full license in the **[LICENSE](LICENSE)** file.

We encourage all modifications and contributions to be shared back with the community, for example through pull requests to this repository.  
We also encourage all derivative works to be publicly available under the **EUPL-1.2 License**.  
At all times must the license terms be followed.

For information regarding how to contribute, see the [contributing](#contributing) section above.

CTF Pilot is owned and maintained by **[The0Mikkel](https://github.com/The0mikkel)**.  
Required Notice: Copyright Mikkel Albrechtsen (<https://themikkel.dk>)

## Code of Conduct

We expect all contributors to adhere to our [Code of Conduct](/CODE_OF_CONDUCT.md) to ensure a welcoming and inclusive environment for all.
