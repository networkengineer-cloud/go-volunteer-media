declare module 'docx-preview' {
  interface RenderOptions {
    className?: string;
    inWrapper?: boolean;
    ignoreWidth?: boolean;
    ignoreHeight?: boolean;
    ignoreFonts?: boolean;
    breakPages?: boolean;
    ignoreLastRenderedPageBreak?: boolean;
    experimental?: boolean;
    trimXmlDeclaration?: boolean;
    debug?: boolean;
    renderHeaders?: boolean;
    renderFooters?: boolean;
    renderFootnotes?: boolean;
    renderEndnotes?: boolean;
    useBase64URL?: boolean;
  }

  export function renderAsync(
    document: Blob | ArrayBuffer,
    bodyContainer: HTMLElement,
    styleContainer?: HTMLElement,
    options?: RenderOptions
  ): Promise<void>;

  export function renderDocx(
    document: Blob | ArrayBuffer,
    bodyContainer: HTMLElement,
    styleContainer?: HTMLElement,
    options?: RenderOptions
  ): Promise<void>;
}