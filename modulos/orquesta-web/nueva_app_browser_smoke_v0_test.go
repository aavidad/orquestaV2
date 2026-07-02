package orquestaweb

import (
	"bytes"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNuevaAppHTMLBrowserSmokeOptInV0(t *testing.T) {
	if strings.TrimSpace(os.Getenv("ORQUESTA_WEB_BROWSER_SMOKE")) != "1" {
		t.Skip("ORQUESTA_WEB_BROWSER_SMOKE=1 no configurado")
	}
	browser := chromiumBrowserForTestV0(t)
	htmlPath := writeNuevaAppBrowserSmokeHTMLForTestV0(t)
	pageURL := (&url.URL{Scheme: "file", Path: htmlPath}).String()

	dom := runChromiumForTestV0(
		t,
		browser,
		"--virtual-time-budget=2500",
		"--window-size=390,900",
		"--dump-dom",
		pageURL,
	)
	if !bytes.Contains(dom, []byte(`data-browser-smoke="ok"`)) ||
		bytes.Contains(dom, []byte("Please fill out this field")) ||
		bytes.Contains(dom, []byte(`data-browser-native-required="true"`)) ||
		bytes.Contains(dom, []byte(`data-browser-horizontal-overflow="true"`)) ||
		bytes.Contains(dom, []byte(`data-browser-expert-storage-anchor="false"`)) ||
		bytes.Contains(dom, []byte(`data-browser-expert-storage-help="false"`)) {
		t.Fatalf("browser smoke fallo\n%s", string(dom))
	}

	for _, viewport := range []struct {
		name   string
		width  string
		height string
	}{
		{name: "mobile", width: "390", height: "900"},
		{name: "desktop", width: "1280", height: "900"},
	} {
		screenshot := filepath.Join(t.TempDir(), "nueva-app-"+viewport.name+".png")
		runChromiumForTestV0(
			t,
			browser,
			"--virtual-time-budget=2500",
			"--window-size="+viewport.width+","+viewport.height,
			"--screenshot="+screenshot,
			pageURL,
		)
		assertPNGNonBlankForTestV0(t, screenshot)
	}
}

func chromiumBrowserForTestV0(t *testing.T) string {
	t.Helper()
	candidates := []string{
		strings.TrimSpace(os.Getenv("CHROMIUM_BIN")),
		"google-chrome",
		"chromium-browser",
		"chromium",
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if path, err := exec.LookPath(candidate); err == nil {
			return path
		}
	}
	t.Skip("Chromium/Chrome no disponible")
	return ""
}

func writeNuevaAppBrowserSmokeHTMLForTestV0(t *testing.T) string {
	t.Helper()
	handler := NewNuevaAppHTMLHandlerV0(&fakeNuevaAppClientV0{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/nueva-app?locale=es", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET nueva-app status=%d body=%s", rec.Code, rec.Body.String())
	}
	html := rec.Body.String()
	marker := `<script>
window.addEventListener('load', function(){
  setTimeout(function(){
    var form = document.querySelector('form[action="/nueva-app"]');
    if (form) {
      form.dispatchEvent(new Event('submit', {bubbles:true, cancelable:true}));
    }
    setTimeout(function(){
      var text = document.body.innerText || '';
      var hasSpanishError = text.indexOf('Completa este campo') !== -1;
      var nativeRequired = !!document.querySelector('[required]');
      var nativeEnglish = text.indexOf(['Please','fill','out','this','field'].join(' ')) !== -1;
      var horizontalOverflow = document.documentElement.scrollWidth > window.innerWidth + 4;
      var storageControl = document.querySelector('[name="datos.storage.5.tipo"]');
      var storageLabel = storageControl ? storageControl.closest('label') : null;
      var storageGuideLink = !!(storageLabel && storageLabel.querySelector('.guide-field-link[href="/nueva-app/guia#modo-experto-preferencia-de-persistencia"]'));
      var storageHelp = !!(storageLabel && storageLabel.querySelector('.help-text'));
      document.body.setAttribute('data-browser-smoke', hasSpanishError && !nativeRequired && !nativeEnglish && !horizontalOverflow && storageGuideLink && storageHelp ? 'ok' : 'failed');
      document.body.setAttribute('data-browser-native-required', String(nativeRequired));
      document.body.setAttribute('data-browser-horizontal-overflow', String(horizontalOverflow));
      document.body.setAttribute('data-browser-expert-storage-anchor', String(storageGuideLink));
      document.body.setAttribute('data-browser-expert-storage-help', String(storageHelp));
    }, 300);
  }, 300);
});
</script>`
	html = strings.Replace(html, "</body>", marker+"</body>", 1)
	path := filepath.Join(t.TempDir(), "nueva-app-browser-smoke.html")
	if err := os.WriteFile(path, []byte(html), 0o600); err != nil {
		t.Fatalf("write html: %v", err)
	}
	return path
}

func runChromiumForTestV0(
	t *testing.T,
	browser string,
	args ...string,
) []byte {
	t.Helper()
	base := []string{
		"--headless=new",
		"--disable-gpu",
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--hide-scrollbars",
	}
	out, err := exec.Command(browser, append(base, args...)...).CombinedOutput()
	if err == nil {
		return out
	}
	base[0] = "--headless"
	out, retryErr := exec.Command(browser, append(base, args...)...).CombinedOutput()
	if retryErr != nil {
		t.Fatalf("chromium fallo: %v; retry=%v; output=%s", err, retryErr, string(out))
	}
	return out
}

func assertPNGNonBlankForTestV0(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read screenshot: %v", err)
	}
	if len(raw) < 1024 {
		t.Fatalf("screenshot demasiado pequena: %s bytes=%d", path, len(raw))
	}
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("decode screenshot: %v", err)
	}
	if distinctSampledColorsForTestV0(img) < 6 {
		t.Fatalf("screenshot aparentemente vacia: %s", path)
	}
}

func distinctSampledColorsForTestV0(img image.Image) int {
	bounds := img.Bounds()
	colors := map[uint32]struct{}{}
	for x := bounds.Min.X; x < bounds.Max.X; x += maxIntForTestV0(1, bounds.Dx()/12) {
		for y := bounds.Min.Y; y < bounds.Max.Y; y += maxIntForTestV0(1, bounds.Dy()/12) {
			r, g, b, a := img.At(x, y).RGBA()
			key := (r>>8)<<24 | (g>>8)<<16 | (b>>8)<<8 | (a >> 8)
			colors[uint32(key)] = struct{}{}
		}
	}
	return len(colors)
}

func maxIntForTestV0(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
