import {Component, OnInit} from 'angular2/core';
import {FileInfo} from '../../models/fileinfo';
import {ApiClient} from '../../services/apiclient';

@Component({
  selector: 'file-list',
  templateUrl: 'app/components/filelist/filelist.html'
})
export class FileListComponent {
  parentId: string
  children: FileInfo[]

  constructor(private _api: ApiClient) {
    this.parentId = "rootNode";
  }

  updateChildren() {
    this._api.listFiles(this.parentId)
      .subscribe((children: FileInfo[]) => {
        this.children = children;
      });
  }

  ngOnInit() {
    this.updateChildren();
  }
}
