window.armoryFace = (() => {
  const base = "/static/face";
  let ready = null;

  const load = () => {
    if (!ready) {
      ready = (async () => {
        await faceapi.tf.ready();
        await faceapi.nets.tinyFaceDetector.loadFromUri(base);
        await faceapi.nets.faceLandmark68TinyNet.loadFromUri(base);
        await faceapi.nets.faceRecognitionNet.loadFromUri(base);
      })();
    }
    return ready;
  };

  const start = async (video) => {
    if (!window.isSecureContext || !navigator.mediaDevices) {
      const host = location.hostname;
      throw new Error(
        "This address is not secure, so the camera is blocked. Install the certificate from http://" + host +
          ":8080/ca.crt, then open https://" + host + ":8443" + location.pathname
      );
    }
    let stream;
    try {
      stream = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: "user", width: { ideal: 640 }, height: { ideal: 480 } },
        audio: false,
      });
    } catch (e) {
      const reasons = {
        NotAllowedError: "Camera access is blocked. Allow the camera in the browser, then reload.",
        NotFoundError: "No camera was found on this device.",
        NotReadableError: "The camera is in use by another app.",
      };
      throw new Error(reasons[e.name] || "The camera could not be started.");
    }
    video.srcObject = stream;
    await video.play();
  };

  const stop = (video) => {
    if (video.srcObject) video.srcObject.getTracks().forEach((t) => t.stop());
    video.srcObject = null;
  };

  const detector = new faceapi.TinyFaceDetectorOptions({ inputSize: 320, scoreThreshold: 0.5 });

  const descriptor = async (video) => {
    const found = await faceapi.detectSingleFace(video, detector).withFaceLandmarks(true).withFaceDescriptor();
    return found ? Array.from(found.descriptor) : null;
  };

  const post = async (url, body) => {
    const res = await fetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  };

  const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

  const watch = (video, handlers) => {
    let stopped = false;
    let busy = false;
    const tick = async () => {
      if (stopped || busy || document.hidden) return;
      busy = true;
      try {
        const d = await descriptor(video);
        if (stopped) return;
        if (!d) {
          handlers.onLooking();
        } else {
          const result = await post("/face/match", { descriptor: d });
          if (stopped) return;
          if (result.matched) {
            stopped = true;
            clearInterval(timer);
            handlers.onMatch(result);
          } else {
            handlers.onNoMatch();
          }
        }
      } catch (e) {
        handlers.onError(e);
      } finally {
        busy = false;
      }
    };
    const timer = setInterval(tick, 600);
    return () => {
      stopped = true;
      clearInterval(timer);
    };
  };

  return { load, start, stop, descriptor, post, sleep, watch };
})();
