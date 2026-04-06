import { useState } from "react";

function ShortenLink() {
    const [link, setLink] = useState("");
    const [shortLink, setShortLink] = useState("");
    const [showPopup, setShowPopup] = useState(false);
    const baseUrl = (import.meta.env.VITE_BASE_URL || "").replace(/\/$/, "");

    const handleGenerateShortLink = () => {
        const requestBody = {
            url: link
        };
        fetch("/api/generate-short-url", {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify(requestBody)
        })
            .then(response => response.json())
            .then(data => {
                setShortLink(`${baseUrl}/link/${data.short_url_code}`);
                setShowPopup(true);
            })
            .catch(error => {
                console.error("Error:", error);
            });
    };

    const handleCopyToClipboard = () => {
        navigator.clipboard.writeText(shortLink);
    };

    const handleClosePopup = () => {
        setShowPopup(false);
        setLink("");
    };

    return (
        <>
            <h1>URL Shortener</h1>
            <div className="wrapper">
                <label className="url-label" htmlFor="originalLink">Original URL</label>
                <input className="input-url" type="text" id="originalLink" value={link} onChange={e => setLink(e.target.value)} /><br/>
                <button className="button-common" onClick={handleGenerateShortLink}>Generate Short URL</button>
            </div>

            {showPopup && (
                <div className="popup">
                    <p>Short URL: {shortLink}</p>
                    <button onClick={handleCopyToClipboard}>Copy to Clipboard</button>
                    <button onClick={handleClosePopup}>Close</button>
                </div>
            )}
        </>
    );
}

export default ShortenLink;

