/*
 _	   __	  _ __
| |	 / /___ _(_) /____
| | /| / / __ `/ / / ___/
| |/ |/ / /_/ / / (__  )
|__/|__/\__,_/_/_/____/
The electron alternative for Go
(c) Lea Anthony 2019-present
*/

import * as Runtime from "../@gailsio/runtime/src";

// NOTE: the following methods MUST be imported explicitly because of how esbuild injection works
import { Enable as EnableWML } from "../@gailsio/runtime/src/wml";
import { debugLog } from "../@gailsio/runtime/src/utils";

window.gails = Runtime;
EnableWML();

if (DEBUG) {
    debugLog("Wails Runtime Loaded")
}
