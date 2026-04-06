export default function Home() {
  return (
    <main className="preview-shell">
      <section className="preview-card" aria-labelledby="preview-title">
        <h1 id="preview-title">Frontend preview</h1>
        <p>
          Đây là không gian xem trước biệt lập cho NPT ShortenLink. Ứng dụng
          React/Vite hiện tại vẫn là frontend production.
        </p>
        <p>
          Luồng rút gọn URL sẽ được kết nối với Python API hiện có trong một
          thay đổi riêng.
        </p>
      </section>
    </main>
  );
}
