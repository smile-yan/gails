import { DockService } from "../bindings/github.com/gailsapp/gails/v3/pkg/services/dock"

const showButton = document.getElementById('show')! as HTMLButtonElement;
const hideButton = document.getElementById('hide')! as HTMLButtonElement;

showButton.addEventListener('click', () => {
    DockService.ShowAppIcon();
});

hideButton.addEventListener('click', () => {
    DockService.HideAppIcon();
});