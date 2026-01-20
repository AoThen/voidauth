import { Component, inject, viewChild } from '@angular/core'
import { MatPaginator } from '@angular/material/paginator'
import { MatSort } from '@angular/material/sort'
import { MatTableDataSource } from '@angular/material/table'
import type { Group } from '@shared/db/Group'
import { AdminService } from '../../../services/admin.service'
import { SnackbarService } from '../../../services/snackbar.service'
import type { TableColumn } from '../clients/clients.component'
import { MaterialModule } from '../../../material-module'
import { ADMIN_GROUP } from '@shared/constants'
import { RouterLink } from '@angular/router'
import { SpinnerService } from '../../../services/spinner.service'
import { MatDialog } from '@angular/material/dialog'
import { ConfirmComponent } from '../../../dialogs/confirm/confirm.component'
import { TranslateModule } from '@ngx-translate/core'
import { I18nService } from '../../../services/i18n.service'

@Component({
  selector: 'app-groups',
  imports: [
    MaterialModule,
    RouterLink,
    TranslateModule,
  ],
  templateUrl: './groups.component.html',
  styleUrl: './groups.component.scss',
})
export class GroupsComponent {
  dataSource: MatTableDataSource<Group> = new MatTableDataSource()

  readonly paginator = viewChild.required(MatPaginator)
  readonly sort = viewChild.required(MatSort)

  columns: TableColumn<Group>[] = [
    {
      columnDef: 'name',
      header: 'Group Name',
      cell: element => element.name,
    },
  ]

  displayedColumns = ([] as string[]).concat(this.columns.map(c => c.columnDef)).concat(['actions'])

  public ADMIN_GROUP = ADMIN_GROUP

  private adminService = inject(AdminService)
  private snackbarService = inject(SnackbarService)
  private spinnerService = inject(SpinnerService)
  private dialog = inject(MatDialog)
  private i18nService = inject(I18nService)

  async ngAfterViewInit() {
    try {
      // Assign the data to the data source for the table to render
      this.spinnerService.show()
      this.dataSource.data = await this.adminService.groups()
      this.dataSource.paginator = this.paginator()
      this.dataSource.sort = this.sort()
    } finally {
      this.spinnerService.hide()
    }
  }

  delete(id: string) {
    const group = this.dataSource.data.find(g => g.id === id)
    const dialogRef = this.dialog.open(ConfirmComponent, {
      data: {
        message: this.i18nService.instant('admin.confirmDeleteGroup', { name: group?.name ?? id }),
        header: this.i18nService.instant('common.delete'),
      },
    })

    dialogRef.afterClosed().subscribe(async (result) => {
      if (!result) {
        return
      }

      try {
        this.spinnerService.show()
        await this.adminService.deleteGroup(id)
        this.dataSource.data = this.dataSource.data.filter(g => g.id !== id)
        this.snackbarService.message(this.i18nService.instant('admin.groupDeleted'))
      } catch (_e) {
        this.snackbarService.error(this.i18nService.instant('admin.errorDeleteGroup'))
      } finally {
        this.spinnerService.hide()
      }
    })
  }
}
