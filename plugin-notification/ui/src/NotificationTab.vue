<template>
  <div class="notification-tab">
    <el-tabs v-model="tabName" type="card">
      <el-tab-pane name="mail" label="Mail">
        <cl-form>
          <cl-form-item :span="2" label="Name" required>
            <!--            <el-input v-model="name" size="small" placeholder="name"/>-->
          </cl-form-item>
        </cl-form>
      </el-tab-pane>
      <el-tab-pane name="mobile" label="Mobile">
        <cl-form :model="form.mobile">
          <cl-form-item :span="4" label="Enabled" prop="enabled">
            <cl-switch v-model="form.mobile.enabled"/>
          </cl-form-item>
          <cl-form-item :span="4" label="Webhook" prop="webhook" :required="form.mobile.enabled">
            <el-input
                v-model="form.mobile.webhook"
                :disabled="!form.mobile.enabled"
                placeholder="Webhook"
                @change="onSave"
            />
          </cl-form-item>
        </cl-form>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script>
import { defineComponent, ref } from 'vue'
import {useRequest} from 'crawlab-ui'

export default defineComponent({
  name: 'NotificationTab',
  setup() {
    const tabName = ref('mail')

    const form = ref({
      mail: {},
      mobile: {},
    })

    const onSave = async() => {
    }

    return {
      tabName,
      form,
      onSave,
    }
  },
})
</script>

<style scoped>
.notification-tab {
  margin: 20px;
}
</style>
