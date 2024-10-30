---
title: Starting Jetify Devspaces
sidebar_position: 2
hide_title: false
---

## From Your Dashboard

If you have not created a Devspace before, you will need to link your Github Account first in order to grant us permissions to access your repositories.

1. Navigate to the [Jetify Dashboard](https://cloud.jetify.com/dashboard) and clicking on the "**Connect Github Repository**" button.
2. After signing in with Github, you will need to give the Jetify Cloud Github App permissions to access and create repositories on your account.

To create a new Jetify Devspace from your [Jetify Dashboard](https://cloud.jetify.com/):

1. Click "**+ Create New**" button on the top right corner of the page.
2. In the modal that appears, enter the URL of the Github repository you want to use for your Devspace.
3. Click "**Create Devspace**" to start your new Devspace

![Create New Devspace](/img/dashboard_create_new_devspace.png)

Once you've created a Devspace, you can access it from the Devspace list in your Dashboard.

## With a Github URL

You can start Jetify Devspace from any Github Repo by prepending the repo URL with:

```bash
https://cloud.jetify.com/new/
```

For example, you can open the Devbox repo in a Jetify Devspace by opening the following URL in your browser:

```bash
https://cloud.jetify.com/new/github.com/jetify-com/devbox
```

## From a Template

A full list of available templates and projects can be found in the [Devbox Examples](/docs/devbox/devbox_examples/) page of our documentation.

## From a Project

If you've already created a project in the Jetify Dashboard, you can start a new Devspace from the project by navigating to the project and clicking on the "**+ Create New**" button. This will create a new Devspace using the project's configuration and secrets.
