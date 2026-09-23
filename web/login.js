document.addEventListener("DOMContentLoaded", async () => {
  try {
    const response = await fetch("/auth/me");
    if (response.ok) {
      const data = await response.json();
      location.replace(data.redirect);
      return;
    }
  } catch (_) {
    // The form remains available when there is no active session.
  }

  document.querySelectorAll("[data-demo-email]").forEach((button) => {
    button.addEventListener("click", () => {
      document.querySelector("#login-email").value = button.dataset.demoEmail;
      document.querySelector("#login-password").value = "demo";
      document.querySelector("#login-form").requestSubmit();
    });
  });

  document.querySelector("#login-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    const submit = event.currentTarget.querySelector("button[type=submit]");
    const error = document.querySelector("#login-error");
    submit.disabled = true;
    submit.textContent = "Signing in…";
    error.textContent = "";
    try {
      const response = await fetch("/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          email: document.querySelector("#login-email").value,
          password: document.querySelector("#login-password").value,
        }),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Sign in failed");
      location.replace(data.redirect);
    } catch (requestError) {
      error.textContent = requestError.message;
      submit.disabled = false;
      submit.innerHTML = "Sign in <span>→</span>";
    }
  });
});
