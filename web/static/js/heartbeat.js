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
        }),
      });

      if (!response.ok) {
        console.warn("Heartbeat failed:", response.status);
      }
    } catch (error) {
      console.warn("Heartbeat error:", error);
    }
  };

  // Send immediate heartbeat on page load
  sendHeartbeat();

  // Start heartbeat interval
  const heartbeatInterval = setInterval(sendHeartbeat, 30000); // 30 second intervals

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
