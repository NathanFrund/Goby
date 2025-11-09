// Presence heartbeat management
document.addEventListener("DOMContentLoaded", function () {
  // Detect client type
  function detectClientType() {
    const ua = navigator.userAgent;
    const isMobile =
      /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(ua);
    const isTablet = /iPad|Android(?=.*\bMobile\b)|Tablet/i.test(ua);
    const isIoT =
      /SmartTV|SmartWatch|IoT|Embedded/i.test(ua) || window.innerWidth < 400;

    if (isIoT) return "iot";
    if (isTablet) return "tablet";
    if (isMobile) return "mobile";
    return "desktop";
  }

  // Generate unique client ID for this session
  const clientId =
    "client_" + Date.now() + "_" + Math.random().toString(36).substr(2, 9);
  const clientType = detectClientType();

  // Configure ping behavior based on client type
  const getPingConfig = (clientType) => {
    const configs = {
      desktop: { interval: 30000, multiplier: 3 }, // 30s, expect within 90s
      mobile: { interval: 60000, multiplier: 5 }, // 1m, expect within 5m
      tablet: { interval: 45000, multiplier: 4 }, // 45s, expect within 3m
      iot: { interval: 300000, multiplier: 10 }, // 5m, expect within 50m
    };
    return configs[clientType] || configs.desktop;
  };

  const pingConfig = getPingConfig(clientType);

  // Function to send heartbeat
  const sendHeartbeat = async () => {
    try {
      const response = await fetch("/app/api/presence/heartbeat", {
        method: "POST",
        headers: {
          "Content-Type": "application/x-www-form-urlencoded",
          "X-Requested-With": "XMLHttpRequest",
        },
        body: new URLSearchParams({
          client_id: clientId,
          client_type: clientType,
          ping_interval_ms: pingConfig.interval,
          timeout_multiplier: pingConfig.multiplier,
        }),
      });

      if (!response.ok) {
        console.warn("Heartbeat failed:", response.status);
      }
    } catch (error) {
      console.warn("Heartbeat error:", error);
    }
  };

  // Wait for full page load before starting heartbeats to avoid race condition
  window.addEventListener("load", function () {
    // Send immediate heartbeat now that page is fully loaded
    sendHeartbeat();

    // Start heartbeat interval using client's configured ping interval
    const heartbeatInterval = setInterval(sendHeartbeat, pingConfig.interval);

    // Mark offline when leaving
    window.addEventListener("beforeunload", function () {
      // Use sendBeacon for reliable delivery during page unload
      navigator.sendBeacon(
        "/app/api/presence/offline",
        new URLSearchParams({
          client_id: clientId,
          client_type: clientType,
        })
      );

      // Clear heartbeat
      clearInterval(heartbeatInterval);
    });
  });
});
