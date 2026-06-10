/*
 _	   __	  _ __
| |	 / /___ _(_) /____
| | /| / / __ `/ / / ___/
| |/ |/ / /_/ / / (__  )
|__/|__/\__,_/_/_/____/
The electron alternative for Go
(c) Lea Anthony 2019-present
*/

/**
 * Updater event name constants.
 *
 * Use these instead of hard-coding string literals when subscribing to
 * updater events from JavaScript:
 *
 *     import { Events, Updater } from "@wailsio/runtime";
 *
 *     Events.On(Updater.Events.UpdateAvailable, (e) => {
 *         console.log("update found:", e.data.version);
 *     });
 *
 *     Events.On(Updater.Events.DownloadProgress, (e) => {
 *         const p = e.data;
 *         console.log(`${p.written} / ${p.total} bytes`);
 *     });
 *
 * Mirrors the Go-side constants in `pkg/updater/events.go` and the
 * user-action constants in `pkg/updater/window_lifecycle.go`. Any
 * changes here must stay in sync with those files — there's an
 * integration test that asserts the strings match.
 */
export const Events = Object.freeze({
    /** A Check round-trip is starting. Payload: null. */
    CheckStarted: "gails:updater:check-started",
    /** Check found a newer release. Payload: Release. */
    UpdateAvailable: "gails:updater:update-available",
    /** Check confirmed the caller is up to date. Payload: null. */
    NoUpdate: "gails:updater:no-update",
    /** Download is starting. Payload: Release. */
    DownloadStarted: "gails:updater:download-started",
    /** Periodic progress tick during download (~10 Hz). Payload: Progress. */
    DownloadProgress: "gails:updater:download-progress",
    /** All bytes are on disk, but verification has not yet started. Payload: Release. */
    DownloadComplete: "gails:updater:download-complete",
    /** Signature / digest verification has started. Payload: Release. */
    Verifying: "gails:updater:verifying",
    /** The Updater is swapping the binary into place. Payload: Release. */
    Installing: "gails:updater:installing",
    /** Update is staged and a restart is pending. Payload: Release. */
    UpdateReady: "gails:updater:update-ready",
    /** Something failed. Payload: ErrorInfo { stage, message, provider }. */
    Error: "gails:updater:error",
    /** Host-side context delivered once per session. Payload: Meta { currentVersion, skippedVersion }. */
    Meta: "gails:updater:meta",

    /** Sub-namespace: user-action events that the UI emits BACK to the host. */
    User: Object.freeze({
        /** User clicked Install on an Available update. */
        Install: "gails:updater:user:install",
        /** User clicked Restart & Apply on a Ready update. */
        Restart: "gails:updater:user:restart",
        /** User clicked Skip This Version. */
        Skip: "gails:updater:user:skip",
        /** User clicked Remind Me Later. */
        Remind: "gails:updater:user:remind",
        /** User clicked Close / Cancel. */
        Cancel: "gails:updater:user:cancel",
    }),

    /** Sub-namespace: framework-internal events the UI emits to coordinate
     *  with the host. Most app code can ignore these. */
    Window: Object.freeze({
        /** The window finished loading and asks the host to replay current state. */
        Ready: "gails:updater:window:ready",
    }),
});
