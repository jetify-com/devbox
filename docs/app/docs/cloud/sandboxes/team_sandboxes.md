---
title: Sharing Config and Secrets with Projects
sidebar_position: 4
hide_title: false
---

**Jetify Projects** are a great way to share configuration and secrets with your team. A project can store repository settings and secrets that are shared across all Cloudboxes and users in the project. For example, you can configure a project for a Backend API with the database parameters, API keys, and other secrets. When a user launches a Sandbox in the project, they will automatically have access to the project's configuration and secrets.

## Creating a Project

To create a project:

1. Navigate to the **Projects** tab in the Jetify Dashboard
2. Click on the `Create New` button on the top right corner of the page
3. In the modal that appears, give the project a name, and then click "Create Project"
4. In the new project, navigate to the Settings tab, and then click "Connect with Github" to connect the project to a Github repository
5. Select the account and repository to link to the project.

Once the project is linked to a repository, developers can automatically create a new Sandbox for that repository by navigating to the project and clicking on the `New Sandbox` button.

## Sharing Secrets across Sandboxes

Jetify Projects can store secrets with Jetify Secrets that are shared across all Cloudboxes in the project. Cloudboxes will automatically use the `dev` namespace when accessing secrets.

To add a secret to a project:

1. Navigate to the project in the Jetify Dashboard
2. Click on the `Secrets` tab
3. In the New Secrets form, provide a key and value for the secret, and then click "Add Secret"

Jetify will automatically add the secrets to any Cloudbox that is launched in the project. Note that if a Cloudbox is currently running when you add a secret, you will need to restart it to access the new secret.

For more information on managing secrets, see the [Jetify Secrets](../../secrets) guide.
