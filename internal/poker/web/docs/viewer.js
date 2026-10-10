(() => {
  const image = document.getElementById("diagram");
  const label = document.getElementById("zoom-level");
  let zoom = 100;

  function setZoom(value) {
    zoom = Math.min(300, Math.max(25, value));
    image.style.setProperty("--diagram-width", `${zoom}vw`);
    label.value = `${zoom}%`;
  }

  document.getElementById("zoom-in").addEventListener("click", () => setZoom(zoom + 25));
  document.getElementById("zoom-out").addEventListener("click", () => setZoom(zoom - 25));
  document.getElementById("zoom-reset").addEventListener("click", () => setZoom(100));
})();
