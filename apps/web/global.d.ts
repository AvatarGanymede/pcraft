export {};

declare global {
  interface Window {
    // Port injection for dev mode (browser on web port, API on backend port)
    __pcraft_API_PORT?: string;
    // Debug mode flag (injected by the Go shell or derived from boot payload runtime config)
    __PCRAFT_DEBUG?: boolean;
  }
}
