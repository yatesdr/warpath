// SSE connection for live updates
(function() {
  let es;

  function connect() {
    es = new EventSource('/events');

    es.addEventListener('order-update', function(e) {
      // Refresh order-related content
      const el = document.querySelector('[data-sse="orders"]');
      if (el) htmx.trigger(el, 'sse-refresh');
      // Also refresh dashboard stats
      const dash = document.querySelector('[data-sse="dashboard"]');
      if (dash) htmx.trigger(dash, 'sse-refresh');
    });

    es.addEventListener('inventory-update', function(e) {
      const el = document.querySelector('[data-sse="nodestate"]');
      if (el) htmx.trigger(el, 'sse-refresh');
    });

    es.addEventListener('node-update', function(e) {
      const el = document.querySelector('[data-sse="nodes"]');
      if (el) htmx.trigger(el, 'sse-refresh');
    });

    es.addEventListener('system-status', function(e) {
      const data = JSON.parse(e.data);
      if (data.fleet !== undefined) {
        const el = document.getElementById('fleet-status');
        if (el) {
          el.className = 'health ' + (data.fleet === 'connected' ? 'health-ok' : 'health-fail');
        }
      }
      if (data.messaging !== undefined) {
        const el = document.getElementById('msg-status');
        if (el) {
          el.className = 'health ' + (data.messaging === 'connected' ? 'health-ok' : 'health-fail');
        }
      }
    });

    es.onerror = function() {
      es.close();
      setTimeout(connect, 3000);
    };
  }

  connect();
})();
