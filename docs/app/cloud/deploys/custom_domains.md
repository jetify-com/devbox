---
title: Setting a Custom Domain for your Project
sidebar_position: 5
---

Jetify Cloud will automatically configure a unique, private domain for previewing your deployed application. For production purposes, you will probably want to add a more user friendly domain to route users to your application. Jetify will also configure and issue an SSL certificate for your domain automatically.

:::info

You will need access to the DNS records for your Domain in order to configure it for Jetify Cloud.

:::

## Adding a Custom Domain

1. In your project on the Jetify Cloud Dashboard, select **Settings**
1. Scroll down to the **Custom Domain** section on the settings page
   ![Custom Domain Section](../../static/img/custom-domain.png)
1. Enter the custom domain name that you would like to use for your project
1. After you click confirm, your custom domain will be set in a pending state. To validate the domain, you will need to add a record to your DNS provider:
   ![Pending custom domain](../../static/img/custom-domain-unknown.png)
1. Once the correct records have been added to your DNS provider, your Custom Domain will display an **Issued** status:
    ![Custom Domain Issued](../../static/img/custom-domain-issued.png)

## Removing a Custom Domain

You can remove a custom domain by clicking the Delete button. This will remove the domain from your project. Note that after removing the domain, you may want to also clean up your DNS records.
