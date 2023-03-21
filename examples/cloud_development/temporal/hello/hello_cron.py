import asyncio
from dataclasses import dataclass
from datetime import timedelta

from temporalio import activity, workflow
from temporalio.client import Client
from temporalio.worker import Worker


@dataclass
class ComposeGreetingInput:
    greeting: str
    name: str


@activity.defn
async def compose_greeting(input: ComposeGreetingInput) -> str:
    return f"{input.greeting}, {input.name}!"


@workflow.defn
class GreetingWorkflow:
    @workflow.run
    async def run(self, name: str) -> None:
        result = await workflow.execute_activity(
            compose_greeting,
            ComposeGreetingInput("Hello", name),
            start_to_close_timeout=timedelta(seconds=10),
        )
        workflow.logger.info("Result: %s", result)


async def main():
    # Start client
    client = await Client.connect("localhost:7233")

    # Run a worker for the workflow
    async with Worker(
        client,
        task_queue="hello-cron-task-queue",
        workflows=[GreetingWorkflow],
        activities=[compose_greeting],
    ):

        print("Running workflow once a minute")

        # While the worker is running, use the client to start the workflow.
        # Note, in many production setups, the client would be in a completely
        # separate process from the worker.
        await client.start_workflow(
            GreetingWorkflow.run,
            "World",
            id="hello-cron-workflow-id",
            task_queue="hello-cron-task-queue",
            cron_schedule="* * * * *",
        )

        # Wait forever
        await asyncio.Future()


if __name__ == "__main__":
    asyncio.run(main())
