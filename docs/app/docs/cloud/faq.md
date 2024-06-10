---
title: Frequently Asked Questions
description: Frequently Asked Questions about Jetify Cloud
---

This doc contains answers to frequently asked questions about Jetify Cloud that are not covered elsewhere in our documentation. If you have a question that isn't covered here, feel free to ask us on our [Discord](https://discord.gg/jetify)

## Do I have to pay to use Jetify Cloud?

Jetify accounts are free for individual developers, and includes access to Jetify Secrets and the Jetify Prebuilt Cache.

Using Jetify with a team requires a paid Jetify Solo, Starter, or Scale-up account. For details on other plans and limits, see our [**pricing**](https://www.jetify.com/cloud/pricing) page.

## How can I share my Jetify Cloud project with other developers?

To share secrets and access to deployments with other team members, you will need to create a new Jetify Starter Team and then invite developers to join your team. See the [cloud dashboard docs](./dashboard/creating_your_team.md) for more details.

## How do Jetify Cloud Plans work?

Jetify Cloud Plans are available for a monthly platform fee, and allow you to share your Jetify Cloud resources with your team, along with increased support levels. All plans include usage credits equal to the monthly platform fee, which are billed at standard usage rates. 

For more details, see our [**pricing**](https://www.jetify.com/cloud/pricing) page.

## Do you offer self-hosted or private instances of Jetify Cloud?

We offer private instances and other features as part of our Enterprise Plan. [Contact us](https://calendly.com/d/3rd-bhp-qym/meet-with-the-jetify-team) so we can build a solution that meets your needs.

## How does pricing for Jetify Deploys work?

Jetify Deploys cost $0.10/vCPU per hour while your deployment is scheduled. If your Deployment is idle for more than 15 minutes, Jetify Cloud will automatically scale down your deployment to zero. You are not charged for usage while your deployment is scaled down.

## How does pricing for Jetify Cache work?

The Jetify Prebuilt Cache is included in your subscription for no additional cost.

Jetify Private Cache costs $0.60/GB of storage per month for the first 50 GB, and $0.50/GB per month after that. Jetify Private cache does not charge for bandwidth or downloads.

## What size instances are available for Jetify Deploys?

You can configure the following instance sizes for your Deployment.

| Instance | CPU | RAM   |
| -------- | --- | ----- |
| XSmall   | 0.1 | 512MB |
| Small    | 0.5 | 1GB   |
| Medium   | 1   | 2GB   |
| Large    | 2   | 4GB   |

## My project needs a custom instance size or scaling policy

We can customize Jetify Deployments for your project's needs. [Contact us](https://calendly.com/d/3rd-bhp-qym/meet-with-the-jetify-team) for help with a customized solution.
