---
title: Managing Devspace
sidebar_position: 3
hide_title: false
---
Devspaces that you've previously created can be managed from the Jetify Cloud dashboard. You can use the dashboard start, stop, and delete your previous Devspace instances.

## Accessing the Dashboard

To access Devspace in your dashboard, navigate to [cloud.jetify.com](https://cloud.jetify.com) and log in with your Jetify account. Once you select an Org, you will can view all the Devspaces that you have created within that org on the org page.

If you've created any projects in the org, you can also navigate to the project and select the Devspace tab to view all your instances in that project.

## Stopping and Starting Devspace

Your Devspaces are automatically stopped after 15 minutes of user inactivity. This is to help you save resources, and avoid being billed for unused CPU time.

If you want to manually stop your instance, click on the `Options` button on the Devspace that you want to stop, and then click on the `Stop` button. Your package store and project data will be saved, so you can quickly resume your work when you restart the Devspace.

:::info
Stopped instances are not billed for CPU usage, but you may be billed for storage usage in excess of the free tier.
:::

To restart your Devspace, click the option button on the cloud box you want to start, and then click the "Start" button. This will relaunch with the Home directory and packages from your previous session.

## Deleting Devspaces

By default, Devspaces are deleted if they are not used for more than 14 consecutive days. This is to help you save storage costs, and avoid being billed for unused disks.

If you are done with a Devspace, you can delete it by clicking on the `Options` button on the instance that you want to delete, and then clicking on the `Delete` button. This will permanently delete all data associated with the Devspace.
