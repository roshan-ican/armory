self.addEventListener("install", () => self.skipWaiting());
self.addEventListener("activate", (event) => event.waitUntil(
  Promise.all([
    self.clients.claim(),
    caches.keys().then((names) => Promise.all(names.map((name) => caches.delete(name)))),
  ])
));
self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  const url = (event.notification.data && event.notification.data.url) || "/admin/requests";
  event.waitUntil(
    self.clients.matchAll({ type: "window", includeUncontrolled: true }).then((list) => {
      const open = list.find((c) => c.url.includes("/admin/"));
      if (open) {
        open.navigate(url).catch(() => {});
        return open.focus();
      }
      return self.clients.openWindow(url);
    })
  );
});
self.addEventListener("fetch", (event) => {
  if (event.request.mode === "navigate") {
    event.respondWith(fetch(new Request(event.request, { cache: "no-store" })));
  }
});
