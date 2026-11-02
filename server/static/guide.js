document.querySelectorAll("[data-copy]").forEach(function (btn) {
    btn.addEventListener("click", function () {
        var pre = btn.parentElement.nextElementSibling;
        var text = pre.innerText;
        function done() {
            btn.textContent = "Copied";
            setTimeout(function () { btn.textContent = "Copy"; }, 1200);
        }
        function fallback() {
            var area = document.createElement("textarea");
            area.value = text;
            area.setAttribute("readonly", "");
            area.style.position = "fixed";
            area.style.left = "-9999px";
            document.body.appendChild(area);
            area.select();
            var ok = false;
            try { ok = document.execCommand("copy"); } catch (e) { ok = false; }
            area.remove();
            if (ok) done();
        }
        if (navigator.clipboard && window.isSecureContext) {
            navigator.clipboard.writeText(text).then(done, fallback);
        } else {
            fallback();
        }
    });
});
