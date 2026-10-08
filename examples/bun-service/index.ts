const port = Number(process.env.PORT ?? 8080);

Bun.serve({
  port,
  fetch() {
    return new Response("Hello from Bun on Pilum!");
  },
});

console.log(`listening on ${port}`);
