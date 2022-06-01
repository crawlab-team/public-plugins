<template>
  <cl-list-layout
      :table-columns="tableColumns"
      :table-data="tableData"
      :table-total="tableTotal"
      :table-pagination="tablePagination"
      :action-functions="actionFunctions"
      :nav-actions="navActions"
  >
    <template #extra>
      <cl-create-edit-dialog
          :visible="dialogVisible"
          width="1024px"
          no-batch
          @close="onDialogClose"
      >
        <NotificationForm
            :form="form"
        />
      </cl-create-edit-dialog>
    </template>
  </cl-list-layout>
</template>

<script lang="ts">
import {defineComponent, computed, ref, h} from 'vue';
import {useRouter} from 'vue-router';
import {useRequest, ClSwitch, ClNavLink} from 'crawlab-ui';
import NotificationForm from './NotificationForm.vue';

const pluginName = 'notification';
const t = (path) => window['_tp'](pluginName, path);
const _t = window['_t'];

const endpoint = '/plugin-proxy/notification/settings';

const {
  getList,
  post,
} = useRequest();

const getDefaultForm = () => {
  return {
    type: 'mail',
    enabled: true,
  };
};

export default defineComponent({
  name: 'NotificationTemplateList',
  components: {NotificationForm},
  setup() {
    const router = useRouter();

    const tableColumns = computed(() => [
      {
        key: 'name',
        label: t('list.table.columns.name'),
        icon: ['fa', 'font'],
        width: '150',
        value: (row) => h(ClNavLink, {
          label: row.name,
          path: `/notifications/${row._id}`,
        }),
      },
      {
        key: 'type',
        label: t('list.table.columns.type'),
        icon: ['fa', 'list'],
        width: '120',
        value: (row) => t(`notifications.type.${row.type}`)
      },
      {
        key: 'enabled',
        label: t('list.table.columns.enabled'),
        icon: ['fa', 'toggle-on'],
        width: '120',
        value: (row) => h(ClSwitch, {
          modelValue: row.enabled,
          onChange: async (value) => {
            if (!row._id) return;
            if (value.enabled) {
              await post(`${endpoint}/${row._id}/disable`);
            } else {
              await post(`${endpoint}/${row._id}/enable`);
            }
          },
        }),
      },
      {
        key: 'description',
        label: t('list.table.columns.description'),
        icon: ['fa', 'comment-alt'],
        width: '800',
      },
      {
        key: 'actions',
        label: _t('components.table.columns.actions'),
        fixed: 'right',
        width: '200',
        buttons: [
          {
            type: 'primary',
            icon: ['fa', 'search'],
            tooltip: _t('common.actions.view'),
            onClick: (row) => {
              router.push(`/notifications/${row._id}`);
            },
          },
          // {
          //   type: 'info',
          //   size: 'mini',
          //   icon: ['fa', 'clone'],
          //   tooltip: 'Clone',
          //   onClick: (row) => {
          //     console.log('clone', row);
          //   }
          // },
          {
            type: 'danger',
            size: 'mini',
            icon: ['fa', 'trash-alt'],
            tooltip: _t('common.actions.delete'),
            disabled: (row) => !!row.active,
            onClick: async (row) => {
              // const res = await ElMessageBox.confirm('Are you sure to delete?', 'Delete');
              // if (res) {
              // await deleteById(row._id as string);
              // }
              // await getList();
            },
          },
        ],
        disableTransfer: true,
      },
    ]);

    const tableData = ref([]);

    const tablePagination = ref({
      page: 1,
      size: 10,
    });

    const tableTotal = ref(0);

    const actionFunctions = ref({
      getList: async () => {
        const res = await getList(`${endpoint}`, {
          ...tablePagination.value,
        });
        if (!res) {
          tableData.value = [];
          tableTotal.value = 0;
        }
        const {data, total} = res;
        tableData.value = data;
        tableTotal.value = total;
      },
    });


    const form = ref(getDefaultForm());

    const dialogVisible = ref(false);

    const navActions = [
      {
        name: 'common',
        children: [
          {
            buttonType: 'label',
            label: t('list.new.label'),
            tooltip: t('list.new.tooltip'),
            icon: ['fa', 'plus'],
            type: 'success',
            onClick: () => {
              form.value = getDefaultForm();
              dialogVisible.value = true;
            }
          }
        ]
      }
    ];

    const onDialogClose = () => {
      dialogVisible.value = false;
      form.value = getDefaultForm();
    };

    return {
      tableColumns,
      tableData,
      tableTotal,
      tablePagination,
      actionFunctions,
      navActions,
      dialogVisible,
      form,
      onDialogClose,
      t,
    };
  },
});
</script>

<style scoped>

</style>
