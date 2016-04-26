const _mode_dir: number = (1 << (32 - 1))

export class NodeInfo {
  Id: string;
  Name: string;
  Size: number;
  Mode: number;

  constructor(_json: any) {
    this.Id = _json.Id;
    this.Name = _json.Name;
    this.Size = _json.Size;
    this.Mode = _json.Mode;
  }

  public isDir(): boolean {
    return (this.Mode & _mode_dir) != 0;
  }

  public iconString(): string {
    if (this.isDir()) {
      return "folder"
    } else {
      return "description"
    }
  }
}
